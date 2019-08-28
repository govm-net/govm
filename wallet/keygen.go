package wallet

// EcdsaKey ecdsa key
type EcdsaKey struct {
}

// GetType get type
func (k *EcdsaKey) GetType() string {
	return "ecdsa"
}

// Verify verify
func (k *EcdsaKey) Verify(data, sig, pubKey []byte) bool {
	return Recover(pubKey, sig, data)
}

// Sign sign data
func (k *EcdsaKey) Sign(data, privKey []byte) []byte {
	return Sign(privKey, data)
}

// GetPublic get public key
func (k *EcdsaKey) GetPublic(privKey []byte) []byte {
	pubKey := GetPublicKey(privKey)
	return PublicKeyToAddress(pubKey, EAddrTypeDefault)
}
