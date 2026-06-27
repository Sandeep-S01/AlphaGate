package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"sentra/internal/config"
	"sentra/internal/marketdata"
)

type Config struct {
	RESTBaseURL string
	WSBaseURL   string
	HTTPClient  *http.Client
	Dialer      *websocket.Dialer
}

type Client struct {
	restBaseURL string
	wsBaseURL   string
	httpClient  *http.Client
	dialer      *websocket.Dialer
}

type ExchangeInfo struct {
	Symbols []SymbolInfo
}

type SymbolInfo struct {
	Symbol              string
	Status              string
	BaseAsset           string
	QuoteAsset          string
	BaseAssetPrecision  int
	QuoteAssetPrecision int
}

type exchangeInfoResponse struct {
	Symbols []symbolResponse `json:"symbols"`
}

type symbolResponse struct {
	Symbol              string `json:"symbol"`
	Status              string `json:"status"`
	BaseAsset           string `json:"baseAsset"`
	QuoteAsset          string `json:"quoteAsset"`
	BaseAssetPrecision  int    `json:"baseAssetPrecision"`
	QuoteAssetPrecision int    `json:"quoteAssetPrecision"`
}

type KlineRequest = marketdata.HistoricalKlineRequest

func NewFromConfig(cfg config.BinanceConfig) *Client {
	return NewClient(Config{
		RESTBaseURL: cfg.RESTBaseURL,
		WSBaseURL:   cfg.WSBaseURL,
	})
}

func NewClient(cfg Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	dialer := cfg.Dialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}

	return &Client{
		restBaseURL: strings.TrimRight(cfg.RESTBaseURL, "/"),
		wsBaseURL:   strings.TrimRight(cfg.WSBaseURL, "/"),
		httpClient:  httpClient,
		dialer:      dialer,
	}
}

func (c *Client) KlineStreamURL(symbol string, interval string) (string, error) {
	if symbol == "" {
		return "", fmt.Errorf("symbol is required")
	}
	if interval == "" {
		return "", fmt.Errorf("interval is required")
	}

	streamName := strings.ToLower(symbol) + "@kline_" + interval
	base, err := url.Parse(c.wsBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse websocket base URL: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + streamName
	return base.String(), nil
}

func (c *Client) GetExchangeInfo(ctx context.Context, symbol string) (ExchangeInfo, error) {
	endpoint, err := url.Parse(c.restBaseURL + "/api/v3/exchangeInfo")
	if err != nil {
		return ExchangeInfo{}, fmt.Errorf("parse exchange info URL: %w", err)
	}

	query := endpoint.Query()
	if symbol != "" {
		query.Set("symbol", strings.ToUpper(symbol))
	}
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return ExchangeInfo{}, fmt.Errorf("create exchange info request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return ExchangeInfo{}, fmt.Errorf("get exchange info: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ExchangeInfo{}, fmt.Errorf("get exchange info returned status %d", response.StatusCode)
	}

	var decoded exchangeInfoResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return ExchangeInfo{}, fmt.Errorf("decode exchange info response: %w", err)
	}

	info := ExchangeInfo{Symbols: make([]SymbolInfo, 0, len(decoded.Symbols))}
	for _, symbol := range decoded.Symbols {
		info.Symbols = append(info.Symbols, SymbolInfo{
			Symbol:              symbol.Symbol,
			Status:              symbol.Status,
			BaseAsset:           symbol.BaseAsset,
			QuoteAsset:          symbol.QuoteAsset,
			BaseAssetPrecision:  symbol.BaseAssetPrecision,
			QuoteAssetPrecision: symbol.QuoteAssetPrecision,
		})
	}

	return info, nil
}

func (c *Client) FetchKlines(ctx context.Context, request marketdata.HistoricalKlineRequest) ([]marketdata.Candle, error) {
	return c.GetKlines(ctx, request)
}

func (c *Client) GetKlines(ctx context.Context, request KlineRequest) ([]marketdata.Candle, error) {
	endpoint, err := url.Parse(c.restBaseURL + "/api/v3/klines")
	if err != nil {
		return nil, fmt.Errorf("parse klines URL: %w", err)
	}

	query := endpoint.Query()
	query.Set("symbol", strings.ToUpper(request.Symbol))
	query.Set("interval", request.Interval)
	if !request.StartTime.IsZero() {
		query.Set("startTime", strconv.FormatInt(request.StartTime.UnixMilli(), 10))
	}
	if !request.EndTime.IsZero() {
		query.Set("endTime", strconv.FormatInt(request.EndTime.UnixMilli(), 10))
	}
	if request.Limit > 0 {
		query.Set("limit", strconv.Itoa(request.Limit))
	}
	endpoint.RawQuery = query.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create klines request: %w", err)
	}

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("get klines: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("get klines returned status %d", response.StatusCode)
	}

	var raw [][]any
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode klines response: %w", err)
	}

	candles := make([]marketdata.Candle, 0, len(raw))
	for _, entry := range raw {
		candle, err := normalizeKlineEntry(strings.ToUpper(request.Symbol), request.Interval, entry)
		if err != nil {
			return nil, err
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

func normalizeKlineEntry(symbol string, interval string, entry []any) (marketdata.Candle, error) {
	if len(entry) < 9 {
		return marketdata.Candle{}, fmt.Errorf("invalid kline entry length %d", len(entry))
	}

	openTime, err := numberMillis(entry[0])
	if err != nil {
		return marketdata.Candle{}, fmt.Errorf("invalid kline open time: %w", err)
	}
	closeTime, err := numberMillis(entry[6])
	if err != nil {
		return marketdata.Candle{}, fmt.Errorf("invalid kline close time: %w", err)
	}
	tradeCount, err := numberMillis(entry[8])
	if err != nil {
		return marketdata.Candle{}, fmt.Errorf("invalid kline trade count: %w", err)
	}

	return marketdata.Candle{
		Exchange:    "binance",
		Symbol:      symbol,
		Interval:    interval,
		OpenTime:    time.UnixMilli(openTime).UTC(),
		CloseTime:   time.UnixMilli(closeTime).UTC(),
		EventTime:   time.UnixMilli(closeTime).UTC(),
		Open:        fmt.Sprint(entry[1]),
		High:        fmt.Sprint(entry[2]),
		Low:         fmt.Sprint(entry[3]),
		Close:       fmt.Sprint(entry[4]),
		Volume:      fmt.Sprint(entry[5]),
		QuoteVolume: fmt.Sprint(entry[7]),
		TradeCount:  tradeCount,
		IsClosed:    true,
	}, nil
}

func numberMillis(value any) (int64, error) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), nil
	case int64:
		return typed, nil
	case json.Number:
		return typed.Int64()
	case string:
		return strconv.ParseInt(typed, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported number type %T", value)
	}
}

func (c *Client) SubscribeKlines(ctx context.Context, symbol string, interval string) (<-chan marketdata.Candle, error) {
	streamURL, err := c.KlineStreamURL(symbol, interval)
	if err != nil {
		return nil, err
	}

	conn, _, err := c.dialer.DialContext(ctx, streamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect binance kline stream: %w", err)
	}
	slog.Info("binance kline websocket connected", "symbol", strings.ToUpper(symbol), "interval", interval, "url", streamURL)

	candles := make(chan marketdata.Candle)
	go func() {
		defer close(candles)
		defer conn.Close()

		go func() {
			<-ctx.Done()
			_ = conn.Close()
		}()

		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				if ctx.Err() == nil {
					slog.Warn("binance kline websocket read failed", "symbol", strings.ToUpper(symbol), "interval", interval, "error", err)
				}
				return
			}

			candle, err := marketdata.ParseBinanceKlineEvent(payload)
			if err != nil {
				slog.Warn("binance kline websocket payload ignored", "symbol", strings.ToUpper(symbol), "interval", interval, "error", err)
				continue
			}
			slog.Debug("binance kline websocket message received", "symbol", candle.Symbol, "interval", candle.Interval, "open_time", candle.OpenTime, "closed", candle.IsClosed)

			select {
			case <-ctx.Done():
				return
			case candles <- candle:
			}
		}
	}()

	return candles, nil
}
