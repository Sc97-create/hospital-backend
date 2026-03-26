package utils

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strings"
	"time"
)

var (
	lowercaseletters = []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '#', '!', '@'}
)

func GenerateLicenseKey(organisationName string, planday int, organisationID string, planspan string, issuedAt time.Time) (string, time.Time) {
	//ORGNAME-PAYLOAD-SIGNATURE
	//HIR-ENCODEDPAYLOAD-SACHIN@123
	header := ""
	if len(organisationName) > 4 {
		header = organisationName[:4]
	} else {
		header = organisationName
	}
	payload := ""
	calTime := time.Time{}
	switch planspan {
	case "month":
		calTime = time.Now().AddDate(0, planday, 0)

	case "year":
		calTime = time.Now().AddDate(planday, 0, 0)
	}
	res := make(map[string]any)
	res["issued_at"] = issuedAt.Unix()
	res["issued_by"] = organisationID
	res["expiry"] = calTime.Unix()
	res["id"] = organisationID
	val, err := json.Marshal(res)
	if err != nil {
		return "", time.Time{}
	}
	payload = base64.RawStdEncoding.EncodeToString(val)
	finalKey := header + "." + payload + "." + generateRandomstring()
	return finalKey, calTime
}

func generateRandomstring() string {
	res := ""
	for i := 0; i < 5; i++ {
		res += string(lowercaseletters[rand.IntN(len(lowercaseletters))])
	}
	return res
}
func CompareLicenseKey(dblicenseKey string, licensekey string) (err error) {
	dbPayload, err := decodeLicense(dblicenseKey)
	if err != nil {
		return
	}
	payload, err := decodeLicense(licensekey)
	if err != nil {
		return
	}
	dbexpiry, _ := dbPayload["expiry"].(float64)
	expiry, _ := payload["expiry"].(float64)
	dbtime := time.Unix(int64(dbexpiry), 0)
	timePresent := time.Unix(int64(expiry), 0)
	if !dbtime.Equal(timePresent) {
		err = errors.New("expiry is tampered")
		return
	}
	//issuer can also be checked
	return
}
func decodeLicense(license string) (map[string]interface{}, error) {

	parts := strings.Split(license, ".")
	if len(parts) < 2 {
		return nil, errors.New("invalid license format")
	}

	// payload is base64 encoded JSON
	payloadBytes, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, err
	}

	return payload, nil
}

//
