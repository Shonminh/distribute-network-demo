package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

func GenNodeId() string {
	// 生成密钥对
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// 获取公钥
	publicKey := privateKey.PublicKey
	// 对公钥进行 SHA-256 哈希
	nodeID := sha256.Sum256(elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y))
	return fmt.Sprintf("%x", nodeID)
}
