package decoder

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
)

func SenlabTBasicDecoder(ctx context.Context, msg []byte, fn func(context.Context, []byte) error) error {

	dm := []struct {
		DevEUI     string  `json:"devEUI"`
		FPort      string  `json:"fPort,omitempty"`
		Latitude   float64 `json:"latitude,omitempty"`
		Longitude  float64 `json:"longitude,omitempty"`
		Rssi       string  `json:"rssi,omitempty"`
		SensorType string  `json:"sensorType,omitempty"`
		Timestamp  string  `json:"timestamp,omitempty"`
		Payload    string  `json:"payload"`
	}{}

	err := json.Unmarshal(msg, &dm)
	if err != nil {
		return err
	}

	var p payload
	for _, d := range dm {

		b, err := hex.DecodeString(d.Payload)
		if err != nil {
			return err
		}

		// | ID(1) | BatteryLevel(1) | Internal(n) | Temp(2)
		// | ID(1) | BatteryLevel(1) | Internal(n) | Temp(2) | Temp(2)
		if len(b) < 4 {
			return errors.New("invalid payload")
		}

		err = decodePayload(b, &p)
		if err != nil {
			return err
		}

		temp := struct {
			Temperature float32 `json:"temperature"`
		}{
			p.Temperature,
		}

		bat := struct {
			BatteryLevel int `json:"battery_level"`
		}{
			p.BatteryLevel,
		}

		pp := &Payload{
			DevEUI:     d.DevEUI,
			FPort:      d.FPort,
			Latitude:   d.Latitude,
			Longitude:  d.Longitude,
			Rssi:       d.Rssi,
			SensorType: d.SensorType,
			Timestamp:  d.Timestamp,
		}
		pp.Measurements = append(pp.Measurements, temp)
		pp.Measurements = append(pp.Measurements, bat)

		r, err := json.Marshal(&pp)
		if err != nil {
			return nil
		}

		err = fn(ctx, r)
		if err != nil {
			return err
		}
	}

	return nil
}

type payload struct {
	ID           int
	BatteryLevel int
	Temperature  float32
}

func decodePayload(b []byte, p *payload) error {
	id := int(b[0])
	if id == 1 {
		err := singleProbe(b, p)
		if err != nil {
			return err
		}
	}
	if id == 12 {
		err := dualProbe(b, p)
		if err != nil {
			return err
		}
	}

	// these values must be ignored since they are sensor reading errors
	if p.Temperature == -46.75 || p.Temperature == 85 {
		return errors.New("sensor reading error")
	}

	return nil
}

func singleProbe(b []byte, p *payload) error {
	var temp int16
	err := binary.Read(bytes.NewReader(b[len(b)-2:]), binary.BigEndian, &temp)
	if err != nil {
		return err
	}

	p.ID = int(b[0])
	p.BatteryLevel = (int(b[1]) * 100) / 254
	p.Temperature = float32(temp) / 16.0

	return nil
}

func dualProbe(b []byte, p *payload) error {
	return errors.New("unsupported dual probe payload")
}
