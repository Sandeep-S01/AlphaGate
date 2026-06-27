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
	OpenTime  int64  `json:"t"`
	CloseTime int64  `json:"T"`
	Symbol    string `json:"s"`
	Interval  string `json:"i"`
	Open      string `json:"o"`
	Close     string `json:"c"`
	High      string `json:"h"`
	Low       string `json:"l"`
	Volume    string `json:"v"`
	IsClosed  bool   `json:"x"`
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
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  event.Kline.Interval,
		OpenTime:  time.UnixMilli(event.Kline.OpenTime).UTC(),
		CloseTime: time.UnixMilli(event.Kline.CloseTime).UTC(),
		EventTime: time.UnixMilli(event.EventTime).UTC(),
		Open:      event.Kline.Open,
		High:      event.Kline.High,
		Low:       event.Kline.Low,
		Close:     event.Kline.Close,
		Volume:    event.Kline.Volume,
		IsClosed:  event.Kline.IsClosed,
	}, nil
}
