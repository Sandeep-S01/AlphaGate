package marketdata

import (
	"encoding/json"
	"fmt"
	"time"
)

type Candle struct {
	Exchange    string
	Symbol      string
	Interval    string
	OpenTime    time.Time
	CloseTime   time.Time
	EventTime   time.Time
	Open        string
	High        string
	Low         string
	Close       string
	Volume      string
	QuoteVolume string
	TradeCount  int64
	IsClosed    bool
}

type binanceKlineEvent struct {
	EventType string       `json:"e"`
	EventTime int64        `json:"E"`
	Symbol    string       `json:"s"`
	Kline     binanceKline `json:"k"`
}

type binanceKline struct {
	OpenTime    int64         `json:"t"`
	CloseTime   int64         `json:"T"`
	Symbol      string        `json:"s"`
	Interval    string        `json:"i"`
	Open        decimalString `json:"o"`
	Close       decimalString `json:"c"`
	High        decimalString `json:"h"`
	Low         decimalString `json:"l"`
	Volume      decimalString `json:"v"`
	QuoteVolume decimalString `json:"q"`
	TradeCount  int64         `json:"n"`
	IsClosed    bool          `json:"x"`
}

type decimalString string

func (d *decimalString) UnmarshalJSON(payload []byte) error {
	var asString string
	if err := json.Unmarshal(payload, &asString); err == nil {
		*d = decimalString(asString)
		return nil
	}

	var asNumber json.Number
	if err := json.Unmarshal(payload, &asNumber); err != nil {
		return err
	}
	*d = decimalString(asNumber.String())
	return nil
}

func ParseBinanceKlineEvent(payload []byte) (Candle, error) {
	var event binanceKlineEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return Candle{}, fmt.Errorf("decode binance kline event: %w", err)
	}

	if event.EventType != "kline" {
		return Candle{}, fmt.Errorf("unsupported binance event type %q", event.EventType)
	}

	symbol := event.Kline.Symbol
	if symbol == "" {
		symbol = event.Symbol
	}

	return Candle{
		Exchange:    "binance",
		Symbol:      symbol,
		Interval:    event.Kline.Interval,
		OpenTime:    time.UnixMilli(event.Kline.OpenTime).UTC(),
		CloseTime:   time.UnixMilli(event.Kline.CloseTime).UTC(),
		EventTime:   time.UnixMilli(event.EventTime).UTC(),
		Open:        string(event.Kline.Open),
		High:        string(event.Kline.High),
		Low:         string(event.Kline.Low),
		Close:       string(event.Kline.Close),
		Volume:      string(event.Kline.Volume),
		QuoteVolume: string(event.Kline.QuoteVolume),
		TradeCount:  event.Kline.TradeCount,
		IsClosed:    event.Kline.IsClosed,
	}, nil
}
