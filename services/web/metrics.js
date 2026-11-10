let latencyChart;
let replicaChart;
let loadTestChart;

function formatTime(timestamp) {
    return new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function wordleDeployment(sample) {
    return (sample.deployments || []).find(deployment => deployment.name === 'wordle-api') || {};
}

function updateCards(latest) {
    const deployment = wordleDeployment(latest);
    const wordle = latest.wordle_api || {};
    const nginx = latest.nginx || {};
    document.getElementById('latency-value').textContent = latest.probe_latency_ms == null ? '--' : `${latest.probe_latency_ms} ms`;
    document.getElementById('replica-value').textContent = `${deployment.ready_replicas || 0} / ${deployment.desired_replicas || 0}`;
    document.getElementById('request-value').textContent = nginx.requests == null ? '--' : nginx.requests.toLocaleString();
    document.getElementById('restart-value').textContent = wordle.restart_count == null ? '--' : wordle.restart_count;
}

function renderCharts(samples) {
    const labels = samples.map(sample => formatTime(sample.timestamp));
    const latency = samples.map(sample => sample.probe_latency_ms ?? null);
    const ready = samples.map(sample => wordleDeployment(sample).ready_replicas || 0);
    const desired = samples.map(sample => wordleDeployment(sample).desired_replicas || 0);

    if (typeof Chart === 'undefined') {
        document.getElementById('status').textContent = 'Chart.js could not be loaded. Check network access to the CDN.';
        document.getElementById('status').className = 'status error';
        return;
    }

    if (latencyChart) latencyChart.destroy();
    latencyChart = new Chart(document.getElementById('latency-chart'), {
        type: 'line',
        data: { labels, datasets: [{ label: 'Latency (ms)', data: latency, borderColor: '#007f7b', backgroundColor: 'rgba(0, 127, 123, .12)', fill: true, tension: .25, spanGaps: true }] },
        options: { maintainAspectRatio: false, animation: false, scales: { y: { beginAtZero: true } } }
    });

    if (replicaChart) replicaChart.destroy();
    replicaChart = new Chart(document.getElementById('replica-chart'), {
        type: 'line',
        data: { labels, datasets: [
            { label: 'Ready', data: ready, borderColor: '#007f7b', backgroundColor: 'transparent', tension: .25 },
            { label: 'Desired', data: desired, borderColor: '#d49a32', backgroundColor: 'transparent', borderDash: [6, 4], tension: 0 }
        ] },
        options: { maintainAspectRatio: false, animation: false, scales: { y: { beginAtZero: true, ticks: { precision: 0 } } } }
    });
}

function renderEvents(events) {
    const list = document.getElementById('event-list');
    list.replaceChildren();
    if (!events.length) {
        const empty = document.createElement('li');
        empty.className = 'empty';
        empty.textContent = 'No events recorded yet.';
        list.appendChild(empty);
        return;
    }
    events.slice().reverse().forEach(event => {
        const item = document.createElement('li');
        const time = document.createElement('span');
        time.className = 'event-time';
        time.textContent = formatTime(event.timestamp);
        item.append(time, `${event.type}: ${event.message}`);
        list.appendChild(item);
    });
}

function renderLoadTests(tests) {
    const status = document.getElementById('test-status');
    const p50 = document.getElementById('test-p50');
    const p95 = document.getElementById('test-p95');
    const success = document.getElementById('test-success');
    const info = document.getElementById('test-info');
    const completed = tests.filter(test => test.status === 'completed');
    const latest = tests[tests.length - 1];

    if (!latest) {
        status.textContent = '--';
        p50.textContent = '--';
        p95.textContent = '--';
        success.textContent = '--';
        info.textContent = 'Run a load test from the CLI to populate this chart.';
        if (loadTestChart) loadTestChart.destroy();
        return;
    }

    status.textContent = latest.status;
    p50.textContent = latest.p50_latency_ms ? `${latest.p50_latency_ms} ms` : '--';
    p95.textContent = latest.p95_latency_ms ? `${latest.p95_latency_ms} ms` : '--';
    const total = latest.successes + latest.errors;
    success.textContent = total ? `${((latest.successes / total) * 100).toFixed(1)}%` : '--';
    info.textContent = `${latest.id} · ${latest.requested_rps} RPS for ${latest.duration_seconds}s · ${latest.requests_sent.toLocaleString()} requests`;

    if (typeof Chart === 'undefined' || !completed.length) return;
    if (loadTestChart) loadTestChart.destroy();
    loadTestChart = new Chart(document.getElementById('load-test-chart'), {
        type: 'line',
        data: {
            labels: completed.map(test => formatTime(test.started_at)),
            datasets: [
                { label: 'p50 (ms)', data: completed.map(test => test.p50_latency_ms), borderColor: '#007f7b', backgroundColor: 'transparent', tension: .2 },
                { label: 'p95 (ms)', data: completed.map(test => test.p95_latency_ms), borderColor: '#d95f59', backgroundColor: 'transparent', tension: .2 }
            ]
        },
        options: { maintainAspectRatio: false, animation: false, scales: { y: { beginAtZero: true } } }
    });
}

async function refresh() {
    const status = document.getElementById('status');
    try {
        const [metricsResponse, eventsResponse, testsResponse] = await Promise.all([
            fetch('/api/v1/metrics?limit=60'),
            fetch('/api/v1/events?limit=10'),
            fetch('/api/v1/load-tests')
        ]);
        if (!metricsResponse.ok || !eventsResponse.ok || !testsResponse.ok) throw new Error('The control plane returned an error.');
        const metrics = await metricsResponse.json();
        const events = await eventsResponse.json();
        const tests = await testsResponse.json();
        if (!metrics.samples.length) {
            status.textContent = metrics.collector_error || 'Waiting for the first metrics sample...';
            status.className = 'status stale';
            renderEvents(events.events || []);
            renderLoadTests(tests || []);
            return;
        }
        const latest = metrics.samples[metrics.samples.length - 1];
        updateCards(latest);
        renderCharts(metrics.samples);
        renderEvents(events.events || []);
        renderLoadTests(tests || []);
        status.textContent = metrics.collector_error ? `Collector warning: ${metrics.collector_error}` : `Updated ${formatTime(latest.timestamp)} · sampling every ${metrics.sample_interval_seconds}s`;
        status.className = metrics.collector_error ? 'status stale' : 'status';
    } catch (error) {
        status.textContent = `Dashboard unavailable: ${error.message}`;
        status.className = 'status error';
    }
}

refresh();
setInterval(refresh, 5000);
