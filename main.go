package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/xssnick/tonutils-go/tvm/cell"
)

type EstimateRequestPayload struct {
	Address      string `json:"address"`
	Body         string `json:"body"`
	IgnoreChksig bool   `json:"ignoreChksig"`
	InitCode     string `json:"initCode"`
	InitData     string `json:"initData"`
}

func createTransactionBody(message string) (string, error) {
	builder := cell.BeginCell()

	err := builder.StoreBinarySnake([]byte(message))
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

	fmt.Printf("Total estimated fee: %.9f TON.\n", totalFeeInNano)

	return totalFeeInNano, nil
}

func main() {
	walletAddress := "EQDJlZqZfh1OQ4PY2ze4bSEBznjc8fGzkE2YiP5XLvDv1M6u"

	transactionBody, err := createTransactionBody("te6ccgECEAEAAigAART/APSkE/S88sgLAQIBIAIDAgFIBAUB9vLUgwjXGNEh+QDtRNDT/9Mf9AT0BNM/0xXR+CMhoVIguY4SM234IySqAKESuZJtMt5Y+CMB3lQWdfkQ8qEG0NMf1NMH0wzTCdM/0xXRUWi68qJRWrrypvgjKqFSULzyowT4I7vyo1MEgA30D2+hmdAk1yHXCgDyZJEw4g4AeNAg10vAAQHAYLCRW+EB0NMDAXGwkVvg+kAw+CjHBbORMODTHwGCEK5C5aS6nYBA1yHXTPgqAe1V+wTgMAIBIAYHAgJzCAkCASAMDQARrc52omhrhf/AAgEgCgsAGqu27UTQgQEi1yHXCz8AGKo77UTQgwfXIdcLHwAbuabu1E0IEBYtch1wsVgA5bi/Ltou37IasJAoQJsO1E0IEBINch9AT0BNM/0xXRBY4b+CMloVIQuZ8ybfgjBaoAFaESuZIwbd6SMDPikjAz4lIwgA30D2+hntAh1yHXCgCVXwN/2zHgkTDiWYAN9A9voZzQAdch1woAk3/bMeCRW+JwgB/lMJgA30D2+hjhPQUATXGNIAAfJkyFjPFs+DAc8WjhAwyCTPQM+DhAlQBaGlFM9A4vgAyUA5gA30FwTIy/8Tyx/0ABL0ABLLPxLLFcntVPgPIdDTAAHyZdMCAXGwkl8D4PpAAdcLAcAA8qX6QDH6ADH0AfoAMfoAMYBg1yHTAAEPACDyZdIAAZPUMdGRMOJysfsA")
	if err != nil {
		log.Fatalf("Error creating transaction body: %v", err)
	}

	fee, err := EstimateFee(walletAddress, transactionBody)
	if err != nil {
		log.Fatalf("Error estimating fee: %v", err)
	}

	fmt.Printf("Estimated fee: %.9f TON\n", fee)
}
