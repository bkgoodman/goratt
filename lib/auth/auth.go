package auth
 
import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)
 
// SignOpenRequest signs a remote open request.
func SignOpenRequest(base64Secret, member, tool string, ts uint64) (string, string, error) {
	secret, err := base64.StdEncoding.DecodeString(base64Secret)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 secret: %w", err)
	}
	if len(secret) == 0 {
		return "", "", fmt.Errorf("secret cannot be empty")
	}
 
	msg := make([]byte, 0, len(member)+len(tool)+8)
	msg = append(msg, []byte(member)...)
	msg = append(msg, []byte(tool)...)
 
	var tsBuf [8]byte
	binary.BigEndian.PutUint64(tsBuf[:], ts)
	msg = append(msg, tsBuf[:]...)
 
	mac := hmac.New(sha256.New, secret)
	mac.Write(msg)
	sum := mac.Sum(nil)
 
	return hex.EncodeToString(sum), base64.StdEncoding.EncodeToString(sum), nil
}
 
// VerifySignature verifies the signature of a remote open request.
func VerifySignature(base64Secret, member, tool string, ts uint64, providedSig string) error {
	sigHex, sigBase64, err := SignOpenRequest(base64Secret, member, tool, ts)
	if err != nil {
		return err
	}
 
	// Try hex
	if decoded, err := hex.DecodeString(providedSig); err == nil {
		expected, _ := hex.DecodeString(sigHex)
		if subtle.ConstantTimeCompare(decoded, expected) == 1 {
			return nil
		}
	}
 
	// Try base64
	if decoded, err := base64.StdEncoding.DecodeString(providedSig); err == nil {
		expected, _ := base64.StdEncoding.DecodeString(sigBase64)
		if subtle.ConstantTimeCompare(decoded, expected) == 1 {
			return nil
		}
	}
 
	return fmt.Errorf("signature verification failed")
}
