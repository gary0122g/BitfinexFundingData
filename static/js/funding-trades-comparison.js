// Global variables
let comparisonChart = null;
let currentPage = 1;
const ITEMS_PER_PAGE = 20;

// Initialize the view
function initFundingTradesComparisonView() {
    // Load initial data
    loadComparisonData();

    // Set up auto-refresh
    setInterval(loadComparisonData, 60000); // Refresh every minute
}

// Load comparison data
function loadComparisonData() {
    const currency = document.getElementById('currency-select').value;
    const limit = document.getElementById('limit-select').value;

    // Show loading state
    document.getElementById('comparison-loading').style.display = 'block';
    document.getElementById('comparison-error').style.display = 'none';

    // First, get the funding stats
    fetch(`/api/funding-stats/${currency}?limit=${limit}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        })
        .then(statsData => {
            console.log('Received funding stats:', statsData); // Debug log

            // Then get the trades data
            return fetch(`/api/funding-trades-comparison/${currency}?limit=${limit}`)
                .then(response => {
                    if (!response.ok) {
                        throw new Error(`HTTP error! status: ${response.status}`);
                    }
                    return response.json();
                })
                .then(tradesData => {
                    // Hide loading state
                    document.getElementById('comparison-loading').style.display = 'none';

                    if (!tradesData || !tradesData.trades) {
                        throw new Error('Invalid trades data received');
                    }

                    console.log('Received trades data:', tradesData); // Debug log

                    // Update chart with both stats and trades
                    updateComparisonChart(statsData, tradesData.trades);

                    // Update table with trades and the latest FRR from stats
                    const latestFRR = statsData[0]?.FRR || statsData[0]?.frr || 0;
                    updateComparisonTable(tradesData.trades, latestFRR);

                    // Update last updated time
                    updateLastUpdated('comparison-last-updated');
                });
        })
        .catch(error => {
            console.error('Error loading comparison data:', error);
            document.getElementById('comparison-loading').style.display = 'none';
            document.getElementById('comparison-error').style.display = 'block';
            document.getElementById('comparison-error').textContent = `Error: ${error.message}`;
        });
}

// Update comparison chart
function updateComparisonChart(stats, trades) {
    // Get the canvas element
    const canvas = document.getElementById('comparison-chart');
    if (!canvas) {
        console.error('Canvas element not found');
        return;
    }

    // Destroy existing chart if it exists
    if (comparisonChart) {
        comparisonChart.destroy();
        comparisonChart = null;
    }

    // Prepare data
    const statsData = stats.map(item => {
        const mts = item.MTS || item.mts;
        const frr = item.FRR || item.frr;
        return {
            x: new Date(mts),
            y: frr * 100 // Convert to percentage (hourly rate)
        };
    }).filter(item => item.y !== null && !isNaN(item.y));

    const tradesData = trades.map(item => {
        const mts = item.MTS || item.mts;
        const rate = item.Rate || item.rate;
        return {
            x: new Date(mts),
            y: rate * 365 * 100 // Convert to APR percentage
        };
    }).filter(item => item.y !== null && !isNaN(item.y));

    console.log('Processed chart data:', { statsData, tradesData }); // Debug log

    // Create new chart
    comparisonChart = new Chart(canvas, {
        type: 'line',
        data: {
            datasets: [{
                label: 'FRR (Hourly %)',
                data: statsData,
                borderColor: 'rgb(75, 192, 192)',
                tension: 0.1,
                fill: false
            }, {
                label: 'Trade Rate (APR %)',
                data: tradesData,
                borderColor: 'rgb(255, 99, 132)',
                tension: 0.1,
                fill: false
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: false,
                    title: {
                        display: true,
                        text: 'Rate (%)'
                    }
                },
                x: {
                    type: 'time',
                    time: {
                        unit: 'hour',
                        displayFormats: {
                            hour: 'MMM d, HH:mm'
                        }
                    },
                    title: {
                        display: true,
                        text: 'Time'
                    }
                }
            },
            plugins: {
                tooltip: {
                    callbacks: {
                        label: function (context) {
                            return `${context.dataset.label}: ${formatNumber(context.parsed.y, 4, true)}`;
                        }
                    }
                }
            }
        }
    });
}

// Update comparison table
function updateComparisonTable(trades, currentFRR) {
    const tableBody = document.getElementById('comparison-table-body');
    tableBody.innerHTML = '';

    if (!trades || trades.length === 0) {
        tableBody.innerHTML = '<tr><td colspan="6" class="text-center">No data available</td></tr>';
        return;
    }

    console.log('Table data:', { trades, currentFRR }); // Debug log

    trades.forEach(trade => {
        const row = document.createElement('tr');

        // Get values with fallback to lowercase properties
        const mts = trade.MTS || trade.mts;
        const rate = trade.Rate || trade.rate;
        const amount = trade.Amount || trade.amount;

        // Convert rates to percentages
        const tradeRate = rate * 365 * 100; // Convert to APR percentage
        const frrRate = currentFRR * 100; // Keep FRR as hourly percentage

        // Calculate deviation
        const deviation = tradeRate - frrRate;
        const deviationPercent = currentFRR !== 0 ? (deviation / frrRate) * 100 : 0;

        // Set row style based on deviation
        if (deviation > 0) {
            row.classList.add('table-success');
        } else if (deviation < 0) {
            row.classList.add('table-danger');
        }

        row.innerHTML = `
            <td>${formatDate(mts)}</td>
            <td>${formatNumber(tradeRate, 4, true)}</td>
            <td>${formatNumber(frrRate, 4, true)}</td>
            <td>${formatNumber(deviation, 4, true)}</td>
            <td>${currentFRR !== 0 ? formatNumber(deviationPercent, 2, true) : 'N/A'}</td>
            <td>${formatNumber(amount, 2)}</td>
        `;

        tableBody.appendChild(row);
    });
}

// Initialize when the page loads
document.addEventListener('DOMContentLoaded', function () {
    initFundingTradesComparisonView();

    // Listen for currency changes
    document.getElementById('currency-select').addEventListener('change', loadComparisonData);

    // Listen for limit changes
    document.getElementById('limit-select').addEventListener('change', loadComparisonData);
});

function loadPDFDistribution() {
    const currency = document.getElementById('currency-select').value;
    const formattedCurrency = currency.startsWith('f') ? currency : 'f' + currency;
    const binCount = document.getElementById('bin-count').value;

    // 顯示載入中狀態
    document.getElementById('pdf-loading').style.display = 'block';
    document.getElementById('pdf-error').style.display = 'none';

    // 調用新的預計算API端點
    fetch(`/api/rate-distribution/${formattedCurrency}?bins=${binCount}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            return response.json();
        })
        .then(distribution => {
            // 隱藏載入中狀態
            document.getElementById('pdf-loading').style.display = 'none';

            // 檢查是否收到有效的分布數據
            if (!distribution) {
                throw new Error('沒有收到分布數據');
            }

            // 如果是臨時響應（服務尚未實作完成）
            if (distribution.message) {
                console.log('Service message:', distribution.message);
                // 暫時顯示訊息
                document.getElementById('pdf-error').style.display = 'block';
                document.getElementById('pdf-error').textContent = `服務正在開發中: ${distribution.message}`;
                return;
            }

            // 檢查是否有PDF數據
            if (!distribution.pdf || !distribution.labels) {
                throw new Error('收到的分布數據格式無效');
            }

            console.log(`收到預計算的分布數據: ${distribution.total_trades} 筆交易`);
            console.log('Labels:', distribution.labels);
            console.log('PDF values:', distribution.pdf);

            // 直接使用預計算的PDF數據
            updatePDFChart(distribution.labels, distribution.pdf);
            updateLastUpdated('pdf-last-updated');

            // 顯示統計資訊
            displayDistributionStats(distribution);
        })
        .catch(error => {
            console.error('加載PDF分佈數據失敗:', error);
            document.getElementById('pdf-loading').style.display = 'none';
            document.getElementById('pdf-error').style.display = 'block';
            document.getElementById('pdf-error').textContent = `載入失敗: ${error.message}`;
        });
}

function processDataInChunks(rates) {
    const binCount = parseInt(document.getElementById('bin-count').value);

    // 使用更高效的方式計算最大最小值
    let minRate = rates[0];
    let maxRate = rates[0];

    // 分批處理數據以避免堆棧溢出
    const chunkSize = 10000;
    for (let i = 0; i < rates.length; i += chunkSize) {
        const chunk = rates.slice(i, i + chunkSize);
        chunk.forEach(rate => {
            if (rate < minRate) minRate = rate;
            if (rate > maxRate) maxRate = rate;
        });
    }

    const binWidth = (maxRate - minRate) / binCount;
    const bins = new Array(binCount).fill(0);
    const labels = new Array(binCount);

    // 生成標籤
    for (let i = 0; i < binCount; i++) {
        const binStart = minRate + i * binWidth;
        labels[i] = `${binStart.toFixed(2)}%`;
    }

    // 分批處理數據填充分箱
    for (let i = 0; i < rates.length; i += chunkSize) {
        const chunk = rates.slice(i, i + chunkSize);
        chunk.forEach(rate => {
            const binIndex = Math.min(Math.floor((rate - minRate) / binWidth), binCount - 1);
            if (binIndex >= 0) bins[binIndex]++;
        });
    }

    // 計算 PDF
    const total = bins.reduce((sum, count) => sum + count, 0);
    const pdf = bins.map(count => count / total);

    // 更新圖表
    updatePDFChart(labels, pdf);
}

function updatePDFChart(labels, pdf) {
    const ctx = document.getElementById('pdf-chart').getContext('2d');

    if (pdfChart) {
        pdfChart.destroy();
    }

    pdfChart = new Chart(ctx, {
        type: 'bar',
        data: {
            labels: labels,
            datasets: [{
                label: '機率密度',
                data: pdf,
                backgroundColor: 'rgba(54, 162, 235, 0.5)',
                borderColor: 'rgba(54, 162, 235, 1)',
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                y: {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: '機率密度'
                    }
                },
                x: {
                    title: {
                        display: true,
                        text: '利率 (APR %)'
                    },
                    ticks: {
                        maxRotation: 45,
                        minRotation: 45,
                        autoSkip: false,
                        maxTicksLimit: 30
                    }
                }
            },
            plugins: {
                tooltip: {
                    callbacks: {
                        label: function (context) {
                            return `機率: ${(context.raw * 100).toFixed(2)}%`;
                        }
                    }
                }
            }
        }
    });
}

// 新增函數：顯示分布統計資訊
function displayDistributionStats(distribution) {
    // 可以添加一個顯示統計資訊的區域
    const statsHtml = `
        <div class="distribution-stats mt-3">
            <small class="text-muted">
                總交易數: ${distribution.total_trades ? distribution.total_trades.toLocaleString() : 'N/A'} | 
                利率範圍: ${distribution.min_rate ? distribution.min_rate.toFixed(2) : 'N/A'}% - ${distribution.max_rate ? distribution.max_rate.toFixed(2) : 'N/A'}% |
                最後更新: ${distribution.last_updated ? new Date(distribution.last_updated).toLocaleString() : '未知'}
            </small>
        </div>
    `;

    // 找到合適的位置插入統計資訊
    const chartContainer = document.querySelector('#pdf-distribution-view .chart-container');
    if (chartContainer) {
        let statsContainer = chartContainer.querySelector('.distribution-stats');
        if (!statsContainer) {
            chartContainer.insertAdjacentHTML('afterend', statsHtml);
        } else {
            statsContainer.outerHTML = statsHtml;
        }
    }
}