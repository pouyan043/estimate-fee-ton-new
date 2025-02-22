package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/tyler-smith/go-bip39"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
	"golang.org/x/crypto/ed25519"
)

// GenerateWalletData generates a wallet's data including public/private key, address, mnemonic, and seed.
func GenerateWalletData() (string, string, string, string, string) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		log.Fatal(err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		log.Fatal(err)
	}

	seed := bip39.NewSeed(mnemonic, "")
	privateKey := ed25519.NewKeyFromSeed(seed[:32])
	publicKey := privateKey.Public().(ed25519.PublicKey)
	address := generateAddressFromPublicKey(publicKey)

	return base64.StdEncoding.EncodeToString(publicKey), base64.StdEncoding.EncodeToString(privateKey), address, mnemonic, string(seed)
}

// generateAddressFromPublicKey generates a wallet address from a public key.
func generateAddressFromPublicKey(pubKey ed25519.PublicKey) string {
	addr := address.NewAddress(0x1, 0x0, pubKey)
	return addr.String()
}

// EstimateFee calls the TON Center API to estimate the total transaction fee.
func EstimateFee(srcAddr, body string) (float64, error) {
	requestPayload := map[string]interface{}{
		"address":      srcAddr,
		"body":         body,
		"ignoreChksig": true,
		"initCode":     "",
		"initData":     "",
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return 0, fmt.Errorf("error marshaling request: %s", err)
	}

	url := "https://toncenter.com/api/v2/estimateFee"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, fmt.Errorf("error creating request: %s", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()

	var respPayload struct {
		Ok     bool `json:"ok"`
		Result struct {
			SourceFees struct {
				InFwdFee   int `json:"in_fwd_fee"`
				StorageFee int `json:"storage_fee"`
				GasFee     int `json:"gas_fee"`
				FwdFee     int `json:"fwd_fee"`
			} `json:"source_fees"`
		} `json:"result"`
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get estimate fee: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&respPayload)
	if err != nil {
		return 0, fmt.Errorf("error decoding response: %s", err)
	}

	if !respPayload.Ok {
		return 0, fmt.Errorf("error from API: estimate fee not found")
	}

	totalFee := float64(respPayload.Result.SourceFees.InFwdFee + respPayload.Result.SourceFees.StorageFee + respPayload.Result.SourceFees.GasFee + respPayload.Result.SourceFees.FwdFee)
	return totalFee / 1000000000, nil
}

// createTransactionBody generates the transaction body (BOC) for fee estimation.
func createTransactionBody(srcAddr, dstAddr string, amount uint64) (string, error) {
	builder := cell.BeginCell()

	if err := builder.StoreSlice([]byte(srcAddr), uint(len(srcAddr)*8)); err != nil {
		return "", fmt.Errorf("error storing source address: %s", err)
	}

	if err := builder.StoreSlice([]byte(dstAddr), uint(len(dstAddr)*8)); err != nil {
		return "", fmt.Errorf("error storing destination address: %s", err)
	}

	if err := builder.StoreUInt(amount, 64); err != nil {
		return "", fmt.Errorf("error storing amount: %s", err)
	}

	cellBody := builder.EndCell()
	bocBytes := cellBody.ToBOC()
	return base64.StdEncoding.EncodeToString(bocBytes), nil
}

// TestTransactionFee generates wallet data and estimates the transaction fee.
func TestTransactionFee() {
	_, _, srcAddr, _, _ := GenerateWalletData() // Source address
	_, _, dstAddr, _, _ := GenerateWalletData() // Destination address

	amount := "100000000" // Example transaction amount
	parsedAmount, err := strconv.ParseUint(amount, 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	transactionBody, err := createTransactionBody(srcAddr, dstAddr, parsedAmount)
	if err != nil {
		log.Fatal(err)
	}

	fee, err := EstimateFee(srcAddr, transactionBody)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Estimated fee: %.9f TON\n", fee)
}

