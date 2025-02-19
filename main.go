package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/tyler-smith/go-bip39"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type EstimateRequestPayload struct {
	Address      string `json:"address"`
	Body         string `json:"body"`
	IgnoreChksig bool   `json:"ignoreChksig"`
	InitCode     string `json:"initCode"`
	InitData     string `json:"initData"`
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func createTransactionBody(message, amount string) (string, error) {
	builder := cell.BeginCell()

	err := builder.StoreBinarySnake([]byte(message + " " + amount))
	if err != nil {
		return "", fmt.Errorf("error storing data in cell: %v", err)
	}

	cellBody := builder.EndCell()
	bocBytes := cellBody.ToBOC()
	bocBase64 := base64.StdEncoding.EncodeToString(bocBytes)
	return bocBase64, nil
}

func EstimateFee(walletAddress, body string) (float64, error) {
	requestPayload := EstimateRequestPayload{
		Address:      walletAddress,
		Body:         body,
		IgnoreChksig: true,
		InitCode:     "",
		InitData:     "",
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
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get estimate fee: %s", resp.Status)
	}
	var respPayload struct {
		Ok     bool `json:"ok"`
		Result struct {
			SourceFees struct {
				InFwdFee   int64 `json:"in_fwd_fee"`
				StorageFee int64 `json:"storage_fee"`
				GasFee     int64 `json:"gas_fee"`
				FwdFee     int64 `json:"fwd_fee"`
			} `json:"source_fees"`
		} `json:"result"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respPayload)
	if err != nil {
		return 0, fmt.Errorf("error decoding response: %s", err)
	}
	if !respPayload.Ok {
		return 0, fmt.Errorf("error estimating fee: invalid response")
	}
	totalFee := float64(respPayload.Result.SourceFees.InFwdFee +
		respPayload.Result.SourceFees.StorageFee +
		respPayload.Result.SourceFees.GasFee +
		respPayload.Result.SourceFees.FwdFee)
	totalFeeInNano := totalFee / 1000000000.0
	return totalFeeInNano, nil
}

func generateAddressFromMnemonic(mnemonic string) string {
	seed := bip39.NewSeed(mnemonic, "")
	address := hex.EncodeToString(seed[:32])
	return address
}

func main() {
	_ = godotenv.Load(".env")
	var walletAddress string
	if _, err := os.Stat(".env"); err == nil {
		walletAddress = os.Getenv("WALLET_ADDRESS")
	} else {
		entropy, err := bip39.NewEntropy(256)
		checkErr(err)
		mnemonic, err := bip39.NewMnemonic(entropy)
		checkErr(err)
		seed := hex.EncodeToString(bip39.NewSeed(mnemonic, ""))
		walletAddress = generateAddressFromMnemonic(mnemonic)
		err = godotenv.Write(map[string]string{
			"MNEMONIC":       mnemonic,
			"SEED":           seed,
			"WALLET_ADDRESS": walletAddress,
		}, ".env")
		checkErr(err)
	}
	transactionBody, err := createTransactionBody("Test transaction message to "+walletAddress, "1000000000")
	checkErr(err)
	fee, err := EstimateFee(walletAddress, transactionBody)
	checkErr(err)
	fmt.Printf("Estimated fee: %.9f TON\n", fee)
}
