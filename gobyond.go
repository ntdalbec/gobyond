package gobyond

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"time"
)

func encode(payload string) []byte {
	request := []byte{0x00, 0x83}
	short := uint16(len(payload) + 6)
	expectedDataLength := make([]byte, 2)
	binary.BigEndian.PutUint16(expectedDataLength, short)
	request = append(request, expectedDataLength...)
	request = append(request, 0x00, 0x00, 0x00, 0x00, 0x00)
	request = append(request, []byte(payload)...)
	request = append(request, 0x00)
	return request
}

func isResponseByond(response []byte) bool {
	return response[0] == 0x00 && response[1] == 0x83
}

func isResponseBodyASCII(response []byte) bool {
	return response[4] == 0x06
}

func isResponseBodyFloat(response []byte) bool {
	return response[4] == 0x06
}

func decodeBigEndianFloat(floatBytes []byte) float32 {
	bits := binary.BigEndian.Uint32(floatBytes)
	return math.Float32frombits(bits)
}

func parseStatus(fields map[string]string) *Status {
	return &Status{
		Version:             fields["version"],
		Mode:                fields["mode"],
		Respawn:             fields["respawn"],
		Enter:               fields["enter"],
		Vote:                fields["vote"],
		Ai:                  fields["ai"],
		Host:                fields["host"],
		RoundID:             fields["round_id"],
		Players:             fields["players"],
		Revision:            fields["revision"],
		RevisionDate:        fields["revision_date"],
		Admins:              fields["admins"],
		GameState:           fields["gamestate"],
		MapName:             fields["map_name"],
		SecurityLevel:       fields["security_level"],
		RoundDuration:       fields["round_duration"],
		TimeDilationCurrent: fields["time_dilation_current"],
		TimeDilationAvg:     fields["time_dilation_avg"],
		TimeDilationAvgSlow: fields["time_dilation_avg_slow"],
		TimeDilationAvgFast: fields["time_dilation_avg_fast"],
		ShuttleMode:         fields["shuttle_mode"],
		ShuttleTimer:        fields["shuttle_timer"],
	}
}

type Client struct {
	Address string
	timeout time.Duration
}

func NewClient(address string, timeout int) Client {
	timeoutString := fmt.Sprintf("%dms", timeout)
	timeoutDuration, _ := time.ParseDuration(timeoutString)
	return Client{address, timeoutDuration} // TODO: add validation
}

func (client Client) Get(query string) (string, error) {
	conn, err := net.Dial("tcp", client.Address)

	if err != nil {
		return "", err
	}
	defer conn.Close()

	request := encode(query)
	_, err = conn.Write(request)
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(client.timeout)
	conn.SetDeadline(deadline)

	res := make([]byte, 1024)
	_, err = conn.Read(res)
	if err != nil {
		return "", err
	}

	if !isResponseByond(res) {
		err = errors.New("Response was not in byond format")
		return "", err
	}

	if isResponseBodyASCII(res) {
		size := binary.BigEndian.Uint16(res[2:4])
		return string(res[5 : 5+size]), nil
	} else if isResponseBodyFloat(res) {
		// TODO: return union type instead of string maybe?
		float := decodeBigEndianFloat(res[5:9])
		floatString := strconv.FormatFloat(float64(float), 'f', -1, 32)
		return floatString, nil
	}

	return "", errors.New("Unknown content identifier")
}

func (client Client) GetStatus() (*Status, error) {
	response, err := client.Get("?status")
	if err != nil {
		return nil, err
	}

	parsed, err := url.ParseQuery(response)

	flattened := make(map[string]string)

	for key, value := range parsed {
		flattened[key] = value[0]
	}

	status := parseStatus(flattened)

	return status, nil
}

type Status struct {
	Version             string
	Mode                string
	Respawn             string
	Enter               string
	Vote                string
	Ai                  string
	Host                string
	RoundID             string
	Players             string
	Revision            string
	RevisionDate        string
	Admins              string
	GameState           string
	MapName             string
	SecurityLevel       string
	RoundDuration       string
	TimeDilationCurrent string
	TimeDilationAvg     string
	TimeDilationAvgSlow string
	TimeDilationAvgFast string
	ShuttleMode         string
	ShuttleTimer        string
}
