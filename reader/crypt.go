package reader

import (
	"crypto/md5"
	"crypto/rc4"
	"encoding/binary"
	"fmt"
)

// Standard PDF padding (section 7.6.3.3 of ISO 32000-1)
var pdfPadding = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

// encryptInfo holds the encryption parameters parsed from the /Encrypt dictionary.
type encryptInfo struct {
	version     int    // /V: 1=RC4 40-bit, 2=RC4 >40-bit, 4=AES or RC4 128-bit
	revision    int    // /R: algorithm revision
	keyLength   int    // in bytes (default 5 for RC4 40-bit)
	ownerHash   []byte // /O value (32 bytes)
	userHash    []byte // /U value (32 bytes)
	permissions int32  // /P value
	fileID      []byte // first element of trailer /ID array
	key         []byte // computed encryption key
}

// isEncrypted returns true if the document has an /Encrypt entry.
func (d *Document) isEncrypted() bool {
	_, ok := d.trailer["Encrypt"]
	return ok
}

// parseEncryptDict parses the /Encrypt dictionary from the trailer.
func (d *Document) parseEncryptDict() (*encryptInfo, error) {
	encObj, ok := d.trailer["Encrypt"]
	if !ok {
		return nil, nil
	}

	resolved, err := d.resolveIfRef(encObj)
	if err != nil {
		return nil, fmt.Errorf("reader: resolving /Encrypt: %w", err)
	}
	encDict, ok := resolved.(Dict)
	if !ok {
		return nil, fmt.Errorf("reader: /Encrypt is not a dictionary")
	}

	info := &encryptInfo{
		version:   1,
		revision:  2,
		keyLength: 5, // 40-bit default
	}

	if v, ok := encDict.GetInt("V"); ok {
		info.version = int(v)
	}
	if r, ok := encDict.GetInt("R"); ok {
		info.revision = int(r)
	}
	if length, ok := encDict.GetInt("Length"); ok {
		info.keyLength = int(length) / 8
	}
	if p, ok := encDict.GetInt("P"); ok {
		info.permissions = int32(p)
	}

	// /O and /U are string values (32 bytes each)
	if o, ok := encDict["O"]; ok {
		if s, ok := o.(String); ok {
			info.ownerHash = s.Value
		}
	}
	if u, ok := encDict["U"]; ok {
		if s, ok := u.(String); ok {
			info.userHash = s.Value
		}
	}

	// File ID from trailer /ID array
	if idObj, ok := d.trailer["ID"]; ok {
		if idArr, ok := idObj.(Array); ok && len(idArr) > 0 {
			if s, ok := idArr[0].(String); ok {
				info.fileID = s.Value
			}
		}
	}

	return info, nil
}

// decrypt attempts to decrypt the document with the given password.
// An empty password is tried first (for owner-only protection).
func (d *Document) decrypt(password string) error {
	info, err := d.parseEncryptDict()
	if err != nil {
		return err
	}
	if info == nil {
		return nil // not encrypted
	}

	// Only support V=1 (RC4 40-bit) and V=2 (RC4 >40-bit) for now
	if info.version > 2 {
		return fmt.Errorf("reader: unsupported encryption version V=%d", info.version)
	}

	// Try user password first
	key := computeEncryptionKey([]byte(password), info)
	if validateUserPassword(key, info) {
		info.key = key
		d.encrypt = info
		return nil
	}

	// Try as owner password
	userPass := recoverUserPassFromOwner([]byte(password), info)
	key = computeEncryptionKey(userPass, info)
	if validateUserPassword(key, info) {
		info.key = key
		d.encrypt = info
		return nil
	}

	return fmt.Errorf("reader: invalid password")
}

// computeEncryptionKey implements Algorithm 2 from the PDF spec.
// Computes the encryption key from the user password.
func computeEncryptionKey(password []byte, info *encryptInfo) []byte {
	// Pad or truncate password to 32 bytes
	padded := make([]byte, 32)
	copy(padded, password)
	if len(password) < 32 {
		copy(padded[len(password):], pdfPadding[:32-len(password)])
	}

	h := md5.New()
	h.Write(padded)
	h.Write(info.ownerHash)

	// Permission bytes (little-endian)
	var pbuf [4]byte
	binary.LittleEndian.PutUint32(pbuf[:], uint32(info.permissions))
	h.Write(pbuf[:])

	h.Write(info.fileID)

	digest := h.Sum(nil)

	// For R >= 3, do 50 additional MD5 iterations
	if info.revision >= 3 {
		for i := 0; i < 50; i++ {
			tmp := md5.Sum(digest[:info.keyLength])
			digest = tmp[:]
		}
	}

	return digest[:info.keyLength]
}

// validateUserPassword checks if the computed key matches the /U value.
// Algorithm 6 (R=2) or Algorithm 7 (R=3+).
func validateUserPassword(key []byte, info *encryptInfo) bool {
	if info.revision == 2 {
		// Algorithm 4: encrypt padding with key
		c, err := rc4.NewCipher(key)
		if err != nil {
			return false
		}
		computed := make([]byte, 32)
		c.XORKeyStream(computed, pdfPadding)
		return bytesEqual(computed, info.userHash)
	}

	// R >= 3: Algorithm 5
	h := md5.New()
	h.Write(pdfPadding)
	h.Write(info.fileID)
	digest := h.Sum(nil)

	// RC4 encrypt with key
	c, err := rc4.NewCipher(key)
	if err != nil {
		return false
	}
	c.XORKeyStream(digest, digest)

	// 19 additional RC4 passes with modified keys
	for i := 1; i <= 19; i++ {
		newKey := make([]byte, len(key))
		for j := range key {
			newKey[j] = key[j] ^ byte(i)
		}
		c, err = rc4.NewCipher(newKey)
		if err != nil {
			return false
		}
		c.XORKeyStream(digest, digest)
	}

	// Compare first 16 bytes
	if len(info.userHash) < 16 || len(digest) < 16 {
		return false
	}
	return bytesEqual(digest[:16], info.userHash[:16])
}

// recoverUserPassFromOwner recovers the user password from the owner password.
// Algorithm 7 from the PDF spec.
func recoverUserPassFromOwner(ownerPass []byte, info *encryptInfo) []byte {
	padded := make([]byte, 32)
	copy(padded, ownerPass)
	if len(ownerPass) < 32 {
		copy(padded[len(ownerPass):], pdfPadding[:32-len(ownerPass)])
	}

	digest := md5.Sum(padded)

	if info.revision >= 3 {
		for i := 0; i < 50; i++ {
			tmp := md5.Sum(digest[:])
			digest = tmp
		}
	}

	key := digest[:info.keyLength]

	userPass := make([]byte, len(info.ownerHash))
	copy(userPass, info.ownerHash)

	if info.revision == 2 {
		c, _ := rc4.NewCipher(key)
		c.XORKeyStream(userPass, userPass)
	} else {
		// R >= 3: 20 RC4 passes in reverse
		for i := 19; i >= 0; i-- {
			newKey := make([]byte, len(key))
			for j := range key {
				newKey[j] = key[j] ^ byte(i)
			}
			c, _ := rc4.NewCipher(newKey)
			c.XORKeyStream(userPass, userPass)
		}
	}

	return userPass
}

// makeObjectCipher creates an RC4 cipher for decrypting strings/streams
// in the given object. The cipher state must be maintained across all
// strings in the same object because gofpdf reuses it during encryption.
func (d *Document) makeObjectCipher(objNum, genNum int) *rc4.Cipher {
	if d.encrypt == nil || d.encrypt.key == nil {
		return nil
	}

	// Per-object key: MD5(fileKey + objNum(3 bytes LE) + genNum(2 bytes LE))
	var buf []byte
	buf = append(buf, d.encrypt.key...)

	var objBuf [4]byte
	binary.LittleEndian.PutUint32(objBuf[:], uint32(objNum))
	buf = append(buf, objBuf[0], objBuf[1], objBuf[2])

	var genBuf [4]byte
	binary.LittleEndian.PutUint32(genBuf[:], uint32(genNum))
	buf = append(buf, genBuf[0], genBuf[1])

	hash := md5.Sum(buf)
	keyLen := len(d.encrypt.key) + 5
	if keyLen > 16 {
		keyLen = 16
	}

	c, _ := rc4.NewCipher(hash[:keyLen])
	return c
}

// bytesEqual compares two byte slices for equality.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		// Compare up to the shorter length
		n := len(a)
		if len(b) < n {
			n = len(b)
		}
		if n == 0 {
			return false
		}
		return bytesEqual(a[:n], b[:n])
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
