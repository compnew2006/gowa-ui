package main

import (
    "fmt"
    "github.com/compnew2006/whatomate/internal/crypto"
)

func main() {
    key := "6205769742c38848480434b9d5f0e9b96431976292724458f237890696328904"
    secret := "tester_secret"
    // Use the Encrypt function directly from the internal/crypto package
    enc, err := crypto.Encrypt(secret, key)
    if err != nil {
        panic(err)
    }
    fmt.Println(enc)
}
