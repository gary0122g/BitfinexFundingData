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