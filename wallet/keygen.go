package wallet

// EcdsaKey ecdsa key
type EcdsaKey struct {
	Type       string
	NeedVerify bool
}

// GetType get type
func (k *EcdsaKey) GetType() string {
	if k.Type != "" {
		return k.Type
	}
	return "ecdsa"
}

// Verify verify
func (k *EcdsaKey) Verify(data, sig, pubKey []byte) bool {
	if !k.NeedVerify {
		return true
	}
	return Recover(pubKey, sig, data)
}

// Sign sign data
func (k *EcdsaKey) Sign(data, privKey []byte) []byte {
	return Sign(privKey, data)
}

// GetPublic get public key
func (k *EcdsaKey) GetPublic(privKey []byte) []byte {
	pubKey := GetPublicKey(privKey, EAddrTypeDefault)
	return PublicKeyToAddress(pubKey, EAddrTypeDefault)
}
