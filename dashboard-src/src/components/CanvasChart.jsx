import React, { useRef, useEffect, useState } from 'react';

export default function CanvasChart({ candles = [], signals = [], activePosition = null, activeSymbol = 'BTCUSDT', liveTick = null, isLoading = false, isDataStale = false }) {
  const containerRef = useRef(null);
  const canvasRef = useRef(null);
  const [dimensions, setDimensions] = useState({ width: 600, height: 400 });
  const [zoomLevel, setZoomLevel] = useState(60); // Number of candles visible
  const [scrollOffset, setScrollOffset] = useState(0); // Index offset from the latest candle
  const [hoveredCandle, setHoveredCandle] = useState(null);
  const [mouseCoord, setMouseCoord] = useState(null);

  const stateRef = useRef({ isDragging: false, dragStart: { x: 0, y: 0 }, scrollStart: 0 });

  // Handle auto-resizing canvas to container bounds
  useEffect(() => {
    if (!containerRef.current) return;
    const resizeObserver = new ResizeObserver((entries) => {
      for (let entry of entries) {
        setDimensions({
          width: Math.floor(entry.contentRect.width),
          height: Math.floor(entry.contentRect.height),
        });
      }
    });
    resizeObserver.observe(containerRef.current);
    return () => resizeObserver.disconnect();
  }, []);

  // Format timestamp to localized standard
  const formatTime = (timeStr) => {
    const d = new Date(timeStr);
    return isNaN(d.getTime()) ? '' : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  };

  const formatDate = (timeStr) => {
    const d = new Date(timeStr);
    return isNaN(d.getTime()) ? '' : d.toLocaleDateString([], { month: 'short', day: 'numeric' });
  };

  // Technical Indicator calculations
  const getEMA = (data, period) => {
    if (data.length < period) return Array(data.length).fill(null);
    const ema = [];
    const k = 2 / (period + 1);
    
    // Calculate simple average for first point
    let sum = 0;
    for (let i = 0; i < period; i++) {
      sum += data[i].close;
    }
    let prevEMA = sum / period;
    
    for (let i = 0; i < data.length; i++) {
      if (i < period - 1) {
        ema.push(null);
      } else if (i === period - 1) {
        ema.push(prevEMA);
      } else {
        const nextEMA = data[i].close * k + prevEMA * (1 - k);
        ema.push(nextEMA);
        prevEMA = nextEMA;
      }
    }
    return ema;
  };

  const getRSI = (data, period = 14) => {
    if (data.length <= period) return Array(data.length).fill(50);
    const rsi = Array(data.length).fill(50);
    let avgGain = 0;
    let avgLoss = 0;

    // Calculate initial average gains/losses (Wilder's exponential smoothing)
    for (let i = 1; i <= period; i++) {
      const change = data[i].close - data[i - 1].close;
      if (change > 0) avgGain += change;
      else avgLoss += Math.abs(change);
    }
    avgGain /= period;
    avgLoss /= period;

    let rs = avgLoss === 0 ? 100 : avgGain / avgLoss;
    rsi[period] = 100 - 100 / (1 + rs);

    for (let i = period + 1; i < data.length; i++) {
      const change = data[i].close - data[i - 1].close;
      const gain = change > 0 ? change : 0;
      const loss = change < 0 ? Math.abs(change) : 0;

      avgGain = (avgGain * (period - 1) + gain) / period;
      avgLoss = (avgLoss * (period - 1) + loss) / period;

      rs = avgLoss === 0 ? 100 : avgGain / avgLoss;
      rsi[i] = 100 - 100 / (1 + rs);
    }
    return rsi;
  };

  // Process data and draw Canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    // Pixel ratio scaling for high-DPI displays
    const dpr = window.devicePixelRatio || 1;
    canvas.width = dimensions.width * dpr;
    canvas.height = dimensions.height * dpr;
    ctx.scale(dpr, dpr);

    // Grid details
    const width = dimensions.width;
    const height = dimensions.height;

    // Sub-chart heights
    const rightScaleWidth = 65;
    const bottomScaleHeight = 22;
    const chartHeight = height - bottomScaleHeight;
    const rsiChartHeight = Math.floor(chartHeight * 0.22);
    const mainChartHeight = chartHeight - rsiChartHeight - 8; // Margin between charts
    const mainChartWidth = width - rightScaleWidth;

    // Clear Screen
    ctx.fillStyle = '#080a12';
    ctx.fillRect(0, 0, width, height);

    if (candles.length === 0) {
      // Draw Grid Placeholder
      ctx.strokeStyle = '#1d2533';
      ctx.lineWidth = 0.5;
      for (let i = 50; i < mainChartWidth; i += 100) {
        ctx.beginPath();
        ctx.moveTo(i, 0);
        ctx.lineTo(i, chartHeight);
        ctx.stroke();
      }
      for (let i = 50; i < chartHeight; i += 50) {
        ctx.beginPath();
        ctx.moveTo(0, i);
        ctx.lineTo(mainChartWidth, i);
        ctx.stroke();
      }
      ctx.fillStyle = '#d8e1f1';
      ctx.font = '12px Inter';
      ctx.textAlign = 'center';
      ctx.fillText('NO MARKET DATA LOADED', mainChartWidth / 2, chartHeight / 2);
      return;
    }

    // Clone and inject live tick price to latest candle if available
    let activeCandles = [...candles];
    if (liveTick && activeCandles.length > 0) {
      const latestIdx = activeCandles.length - 1;
      const latest = activeCandles[latestIdx];
      const nextClose = Number(liveTick);
      activeCandles[latestIdx] = {
        ...latest,
        close: nextClose,
        high: Math.max(latest.high, nextClose),
        low: Math.min(latest.low, nextClose),
      };
    }

    // Precalculate Indicators
    const ema9 = getEMA(activeCandles, 9);
    const ema21 = getEMA(activeCandles, 21);
    const rsiValues = getRSI(activeCandles, 14);

    // Calculate view bounds
    const totalCandles = activeCandles.length;
    let endIdx = totalCandles - scrollOffset;
    let startIdx = endIdx - zoomLevel;

    // Boundary corrections
    if (startIdx < 0) {
      startIdx = 0;
    }
    if (endIdx > totalCandles) {
      endIdx = totalCandles;
    }

    const visibleCandlesCount = endIdx - startIdx;
    const candleWidth = mainChartWidth / visibleCandlesCount;

    // Find min/max prices & volume for scaling
    let minPrice = Infinity;
    let maxPrice = -Infinity;
    let maxVol = 0;

    for (let i = startIdx; i < endIdx; i++) {
      const c = activeCandles[i];
      if (c.high > maxPrice) maxPrice = c.high;
      if (c.low < minPrice) minPrice = c.low;
      if (c.volume > maxVol) maxVol = c.volume;
    }

    // Padding values
    const priceRange = maxPrice - minPrice;
    const padding = priceRange * 0.08 || 1;
    minPrice -= padding;
    maxPrice += padding;

    // Coordinate Conversion Helpers
    const getX = (idx) => (idx - startIdx) * candleWidth + candleWidth / 2;
    const getY = (val) => mainChartHeight - ((val - minPrice) / (maxPrice - minPrice)) * mainChartHeight;
    const getRsiY = (val) => {
      const rsiTop = mainChartHeight + 8;
      const pct = (100 - val) / 100;
      return rsiTop + pct * rsiChartHeight;
    };

    // Draw main chart border dividers
    ctx.strokeStyle = '#3a4659';
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(mainChartWidth, 0);
    ctx.lineTo(mainChartWidth, chartHeight);
    ctx.moveTo(0, mainChartHeight);
    ctx.lineTo(mainChartWidth, mainChartHeight);
    ctx.moveTo(0, mainChartHeight + 8);
    ctx.lineTo(mainChartWidth, mainChartHeight + 8);
    ctx.moveTo(0, chartHeight);
    ctx.lineTo(mainChartWidth, chartHeight);
    ctx.stroke();

    // Draw Price Grid Lines
    ctx.strokeStyle = '#1d2533';
    ctx.lineWidth = 0.5;
    ctx.fillStyle = '#d8e1f1';
    ctx.font = '10px JetBrains Mono';
    ctx.textAlign = 'left';

    const priceGridSteps = 5;
    for (let i = 0; i <= priceGridSteps; i++) {
      const val = minPrice + (priceRange + 2 * padding) * (i / priceGridSteps);
      const y = getY(val);
      if (y >= 0 && y <= mainChartHeight) {
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(mainChartWidth, y);
        ctx.stroke();

        ctx.fillText(val.toFixed(2), mainChartWidth + 6, y + 4);
      }
    }

    // Draw RSI Grid Lines (30, 50, 70)
    ctx.fillStyle = '#d8e1f1';
    ctx.strokeStyle = '#1d2533';
    [30, 50, 70].forEach((lvl) => {
      const rsiY = getRsiY(lvl);
      ctx.beginPath();
      ctx.moveTo(0, rsiY);
      ctx.lineTo(mainChartWidth, rsiY);
      ctx.stroke();

      ctx.fillText(String(lvl), mainChartWidth + 6, rsiY + 4);
    });

    // Draw RSI Oversold/Overbought zones shading
    ctx.fillStyle = 'rgba(124, 92, 255, 0.08)';
    const rsi70y = getRsiY(70);
    const rsi30y = getRsiY(30);
    ctx.fillRect(0, rsi70y, mainChartWidth, rsi30y - rsi70y);

    // Draw Candles & Volume
    for (let i = startIdx; i < endIdx; i++) {
      const c = activeCandles[i];
      const x = getX(i);
      const o = getY(c.open);
      const cl = getY(c.close);
      const h = getY(c.high);
      const l = getY(c.low);
      const isUp = c.close >= c.open;
      const color = isUp ? '#20f2a3' : '#ff5c7a';

      // 1. Draw Volume bar (drawn behind candle wicks)
      const volHeight = (c.volume / maxVol) * (mainChartHeight * 0.15);
      ctx.fillStyle = isUp ? 'rgba(32, 242, 163, 0.22)' : 'rgba(255, 92, 122, 0.22)';
      ctx.fillRect(x - candleWidth * 0.35, mainChartHeight - volHeight, candleWidth * 0.7, volHeight);

      // 2. Draw Wick
      ctx.strokeStyle = color;
      ctx.lineWidth = 1.2;
      ctx.beginPath();
      ctx.moveTo(x, h);
      ctx.lineTo(x, l);
      ctx.stroke();

      // 3. Draw Body
      ctx.fillStyle = color;
      const rectY = Math.min(o, cl);
      const rectH = Math.max(Math.abs(o - cl), 1.5);
      ctx.fillRect(x - candleWidth * 0.35, rectY, candleWidth * 0.7, rectH);

      // Draw Time scales grid intervals
      if (i % Math.ceil(visibleCandlesCount / 5) === 0) {
        ctx.strokeStyle = '#1d2533';
        ctx.beginPath();
        ctx.moveTo(x, 0);
        ctx.lineTo(x, chartHeight);
        ctx.stroke();

        ctx.fillStyle = '#d8e1f1';
        ctx.font = '10px JetBrains Mono';
        ctx.textAlign = 'center';
        ctx.fillText(formatTime(c.openTime), x, chartHeight + 15);
      }
    }

    // Plot EMA Lines
    const drawLineSeries = (dataArray, color) => {
      ctx.strokeStyle = color;
      ctx.lineWidth = 1.2;
      ctx.beginPath();
      let first = true;
      for (let i = startIdx; i < endIdx; i++) {
        const val = dataArray[i];
        if (val === null) continue;
        const x = getX(i);
        const y = getY(val);
        if (first) {
          ctx.moveTo(x, y);
          first = false;
        } else {
          ctx.lineTo(x, y);
        }
      }
      ctx.stroke();
    };

    drawLineSeries(ema9, '#5aa7ff');  // EMA 9 (Stitch primary)
    drawLineSeries(ema21, '#ff5c7a'); // EMA 21 (Stitch tertiary)

    // Plot RSI Oscillator Line
    ctx.strokeStyle = '#7c5cff'; // RSI (Stitch Brand)
    ctx.lineWidth = 1.2;
    ctx.beginPath();
    let firstRsi = true;
    for (let i = startIdx; i < endIdx; i++) {
      const val = rsiValues[i];
      const x = getX(i);
      const rsiY = getRsiY(val);
      if (firstRsi) {
        ctx.moveTo(x, rsiY);
        firstRsi = false;
      } else {
        ctx.lineTo(x, rsiY);
      }
    }
    ctx.stroke();

    // Plot Strategy Signals Arrows & Markers
    signals.forEach((sig) => {
      const sigTime = new Date(sig.generatedAt).getTime();
      const matchIdx = activeCandles.findIndex((c) => new Date(c.openTime).getTime() === sigTime);
      if (matchIdx >= startIdx && matchIdx < endIdx) {
        const x = getX(matchIdx);
        const c = activeCandles[matchIdx];
        const isBuy = sig.side.toLowerCase() === 'buy';

        ctx.fillStyle = isBuy ? '#20f2a3' : '#ff5c7a';
        ctx.beginPath();
        if (isBuy) {
          const arrowY = getY(c.low) + 12;
          ctx.moveTo(x, arrowY);
          ctx.lineTo(x - 5, arrowY + 8);
          ctx.lineTo(x + 5, arrowY + 8);
          ctx.closePath();
          ctx.fill();
          ctx.font = '8px Inter';
          ctx.textAlign = 'center';
          ctx.fillText('BUY', x, arrowY + 18);
        } else {
          const arrowY = getY(c.high) - 12;
          ctx.moveTo(x, arrowY);
          ctx.lineTo(x - 5, arrowY - 8);
          ctx.lineTo(x + 5, arrowY - 8);
          ctx.closePath();
          ctx.fill();
          ctx.font = '8px Inter';
          ctx.textAlign = 'center';
          ctx.fillText('SELL', x, arrowY - 14);
        }
      }
    });

    // Plot Active Position Lines & Entry target annotations
    if (activePosition && activePosition.entryPrice) {
      const entryPrice = Number(activePosition.entryPrice);
      if (entryPrice >= minPrice && entryPrice <= maxPrice) {
        const y = getY(entryPrice);
        ctx.strokeStyle = 'rgba(90, 167, 255, 0.65)';
        ctx.lineWidth = 1;
        ctx.setLineDash([4, 4]);
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(mainChartWidth, y);
        ctx.stroke();
        ctx.setLineDash([]); // Reset

        // Entry Label badge
        ctx.fillStyle = '#1d2533';
        ctx.fillRect(6, y - 8, 100, 16);
        ctx.strokeStyle = '#7c5cff';
        ctx.lineWidth = 0.5;
        ctx.strokeRect(6, y - 8, 100, 16);

        ctx.fillStyle = '#7c5cff';
        ctx.font = '9px JetBrains Mono';
        ctx.textAlign = 'left';
        ctx.fillText(`POS ${activePosition.side} @ ${entryPrice.toFixed(2)}`, 10, y + 3);
      }
    }

    // Draw Crosshair Overlay & Price scale tooltips
    if (mouseCoord && mouseCoord.x >= 0 && mouseCoord.x <= mainChartWidth) {
      const mx = mouseCoord.x;
      const my = mouseCoord.y;

      // Draw dashed crosshair lines
      ctx.strokeStyle = '#d8e1f1';
      ctx.lineWidth = 0.8;
      ctx.setLineDash([3, 3]);

      // Vertical line
      ctx.beginPath();
      ctx.moveTo(mx, 0);
      ctx.lineTo(mx, chartHeight);
      ctx.stroke();

      // Horizontal line
      if (my >= 0 && my <= chartHeight) {
        ctx.beginPath();
        ctx.moveTo(0, my);
        ctx.lineTo(mainChartWidth, my);
        ctx.stroke();
      }
      ctx.setLineDash([]); // Reset

      // Find hovered candle properties
      const hoveredIdx = startIdx + Math.floor(mx / candleWidth);
      if (hoveredIdx >= startIdx && hoveredIdx < endIdx) {
        const hc = activeCandles[hoveredIdx];
        setHoveredCandle({
          ...hc,
          ema9: ema9[hoveredIdx],
          ema21: ema21[hoveredIdx],
          rsi: rsiValues[hoveredIdx],
        });

        // Date tooltip at the bottom scale
        ctx.fillStyle = '#1d2533';
        ctx.fillRect(mx - 55, chartHeight + 2, 110, 18);
        ctx.strokeStyle = '#3a4659';
        ctx.strokeRect(mx - 55, chartHeight + 2, 110, 18);
        ctx.fillStyle = '#f2f6ff';
        ctx.font = '9px JetBrains Mono';
        ctx.textAlign = 'center';
        ctx.fillText(`${formatDate(hc.openTime)} ${formatTime(hc.openTime)}`, mx, chartHeight + 14);
      }

      // Price tooltip at the right scale
      if (my >= 0 && my <= mainChartHeight) {
        const mousePrice = minPrice + ((mainChartHeight - my) / mainChartHeight) * (maxPrice - minPrice);
        ctx.fillStyle = '#1d2533';
        ctx.fillRect(mainChartWidth + 1, my - 9, rightScaleWidth - 2, 18);
        ctx.strokeStyle = '#7c5cff';
        ctx.strokeRect(mainChartWidth + 1, my - 9, rightScaleWidth - 2, 18);
        ctx.fillStyle = '#7c5cff';
        ctx.font = '9px JetBrains Mono';
        ctx.textAlign = 'left';
        ctx.fillText(mousePrice.toFixed(2), mainChartWidth + 6, my + 3);
      }
    } else {
      // Defaults to the latest candle if not hovering
      if (activeCandles.length > 0) {
        const latestIdx = endIdx - 1;
        setHoveredCandle({
          ...activeCandles[latestIdx],
          ema9: ema9[latestIdx],
          ema21: ema21[latestIdx],
          rsi: rsiValues[latestIdx],
        });
      }
    }
  }, [candles, signals, activePosition, activeSymbol, liveTick, dimensions, zoomLevel, scrollOffset, mouseCoord]);

  // Mouse Interaction handlers for zooming and panning
  const handleMouseDown = (e) => {
    if (e.button !== 0) return; // Only left-click
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    stateRef.current.isDragging = true;
    stateRef.current.dragStart = { x, y };
    stateRef.current.scrollStart = scrollOffset;
  };

  const handleMouseMove = (e) => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const rect = canvas.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    setMouseCoord({ x, y });

    if (stateRef.current.isDragging) {
      const dx = x - stateRef.current.dragStart.x;
      const rightScaleWidth = 65;
      const mainChartWidth = dimensions.width - rightScaleWidth;
      const candleWidth = mainChartWidth / zoomLevel;
      const diffIdx = Math.round(dx / candleWidth);
      
      let nextOffset = stateRef.current.scrollStart + diffIdx;
      // Clamping scroll limits
      const maxOffset = candles.length - zoomLevel;
      if (nextOffset < 0) nextOffset = 0;
      if (nextOffset > maxOffset) nextOffset = maxOffset;

      setScrollOffset(nextOffset);
    }
  };

  const handleMouseUpOrLeave = () => {
    stateRef.current.isDragging = false;
    if (mouseCoord) setMouseCoord(null);
  };

  const handleWheel = (e) => {
    e.preventDefault();
    const zoomFactor = e.deltaY > 0 ? 1 : -1;
    let nextZoom = zoomLevel + zoomFactor * Math.ceil(zoomLevel * 0.08);
    
    // Limits: min 15 candles, max 220 candles
    if (nextZoom < 15) nextZoom = 15;
    if (nextZoom > 220) nextZoom = 220;
    if (nextZoom > candles.length) nextZoom = candles.length;

    setZoomLevel(nextZoom);
  };

  // Extract variables for display text overlays
  const dispCandle = hoveredCandle || (candles.length > 0 ? candles[candles.length - 1] : null);

  return (
    <div
      ref={containerRef}
      className={`w-full h-full relative select-none cursor-crosshair ${isLoading ? 'chart-skeleton' : ''}`}
      role="img"
      aria-label={`${activeSymbol} price chart${isDataStale ? ', data feed stale' : ''}`}
      aria-busy={isLoading ? 'true' : 'false'}
    >
      {/* Top Left info overlay */}
      {dispCandle && (
        <div className="absolute top-2 left-3 z-10 font-mono-data text-[10px] text-chrome-text bg-bg-60-1/80 px-2 py-1 rounded border border-chrome-border flex flex-wrap gap-x-3 gap-y-1">
          <span className="text-signal-brand font-bold">{activeSymbol}</span>
          <span>O: <b className="text-white">{Number(dispCandle.open).toFixed(2)}</b></span>
          <span>H: <b className="text-signal-buy">{Number(dispCandle.high).toFixed(2)}</b></span>
          <span>L: <b className="text-signal-sell">{Number(dispCandle.low).toFixed(2)}</b></span>
          <span>C: <b className="text-white">{Number(dispCandle.close).toFixed(2)}</b></span>
          <span>V: <b className="text-chrome-text">{Number(dispCandle.volume).toLocaleString(undefined, { maximumFractionDigits: 0 })}</b></span>
          {dispCandle.ema9 !== undefined && dispCandle.ema9 !== null && (
            <span className="text-signal-brand">EMA9: {dispCandle.ema9.toFixed(2)}</span>
          )}
          {dispCandle.ema21 !== undefined && dispCandle.ema21 !== null && (
            <span className="text-signal-warn">EMA21: {dispCandle.ema21.toFixed(2)}</span>
          )}
          {dispCandle.rsi !== undefined && dispCandle.rsi !== null && (
            <span className="text-signal-info">RSI(14): {dispCandle.rsi.toFixed(2)}</span>
          )}
        </div>
      )}

      <canvas
        ref={canvasRef}
        className={`block w-full h-full ${isLoading ? 'opacity-0' : 'opacity-100'}`}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUpOrLeave}
        onMouseLeave={handleMouseUpOrLeave}
        onWheel={handleWheel}
      />
      {isLoading && (
        <div className="absolute inset-0 flex items-center justify-center text-[11px] uppercase tracking-wider text-chrome-text font-bold">
          Loading chart data
        </div>
      )}
    </div>
  );
}
