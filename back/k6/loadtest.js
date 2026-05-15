import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Trend, Counter, Rate } from 'k6/metrics';
import { randomItem, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

const feedDuration = new Trend('feed_duration_ms', true);
const pollDuration = new Trend('poll_duration_ms', true);
const voteDuration = new Trend('vote_duration_ms', true);
const analyticsDuration = new Trend('analytics_duration_ms', true);
const errors = new Rate('errors');
const httpErrorsByCode = new Counter('http_errors_by_code');

const BASE_URL = 'http://localhost:8080';

export const options = {
  stages: [
    { duration: '1m',  target: 20  },
    { duration: '2m',  target: 50  },
    { duration: '2m',  target: 100 },
    { duration: '2m',  target: 200 },
    { duration: '1m',  target: 0   },
  ],
  thresholds: {
    'feed_duration_ms':       ['p(95)<500'],
    'poll_duration_ms':       ['p(95)<300'],
    'vote_duration_ms':       ['p(95)<500'],
    'analytics_duration_ms':  ['p(95)<300'],
    'errors':                 ['rate<0.05'],
  },
  noConnectionReuse: false,
};

export function setup() {
  const ts = Date.now();
  const email = `loadtest_${ts}@test.ru`;
  const password = 'test123';

  // Register or login
  let res = http.post(`${BASE_URL}/v1/auth/register`, JSON.stringify({
    email: email, password: password, nickname: `tester_${ts}`,
    country: 'RU', gender: 'male', birthYear: 1995,
  }), { headers: { 'Content-Type': 'application/json' } });

  let ok = check(res, { 'register OK': (r) => r.status === 200 });
  let tokens;
  if (ok) {
    tokens = res.json().tokens;
  } else {
    res = http.post(`${BASE_URL}/v1/auth/login`, JSON.stringify({
      email: email, password: password,
    }), { headers: { 'Content-Type': 'application/json' } });
    check(res, { 'login OK': (r) => r.status === 200 });
    tokens = res.json().tokens;
  }

  const accessToken = tokens.accessToken;

  // Create tags with unique names (timestamp suffix)
  const tagNames = [];
  const suffixes = ['aa', 'bb', 'cc', 'dd', 'ee'];
  for (const s of suffixes) {
    const name = `tag_${ts}_${s}`;
    res = http.post(`${BASE_URL}/v1/tags`, JSON.stringify({ name: name }), {
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${accessToken}` },
    });
    if (res.status === 200) {
      tagNames.push(name);
    }
  }
  console.log(`Created ${tagNames.length} tags`);

  if (tagNames.length === 0) {
    console.error('Failed to create any tags, aborting');
    return null;
  }

  // Create 25 polls
  const pollIds = [];
  for (let i = 1; i <= 25; i++) {
    const pollType = i % 3 === 0 ? 'POLL_TYPE_MULTIPLE_CHOICE' : 'POLL_TYPE_SINGLE_CHOICE';
    const pollTags = [tagNames[i % tagNames.length]];
    if (i % 4 === 0) pollTags.push(tagNames[(i + 1) % tagNames.length]);

    res = http.post(`${BASE_URL}/v1/polls`, JSON.stringify({
      question: `Poll #${i}: opinion about ${pollTags[0]}?`,
      type: pollType,
      options: [`Option A${i}`, `Option B${i}`, `Option C${i}`, `Option D${i}`],
      tags: pollTags,
      imageUrl: '',
    }), {
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${accessToken}` },
    });

    if (res.status === 200) {
      pollIds.push(res.json().poll.id);
    }
  }
  console.log(`Created ${pollIds.length} polls`);

  // Cast initial votes (10 out of 25)
  let votesCast = 0;
  for (let i = 0; i < Math.min(pollIds.length, 10); i++) {
    res = http.get(`${BASE_URL}/v1/polls/${pollIds[i]}`, {
      headers: { 'Authorization': `Bearer ${accessToken}` },
    });
    if (res.status === 200) {
      const poll = res.json().poll;
      if (poll.options && poll.options.length > 0) {
        const count = poll.type === 'POLL_TYPE_MULTIPLE_CHOICE' ? 2 : 1;
        const optionIds = poll.options.slice(0, count).map(o => o.id);
        res = http.post(`${BASE_URL}/v1/polls/${pollIds[i]}/vote`, JSON.stringify({
          pollId: pollIds[i],
          optionIds: optionIds,
        }), {
          headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${accessToken}` },
        });
        if (res.status === 200) votesCast++;
      }
    }
  }
  console.log(`Cast ${votesCast} initial votes`);

  return { accessToken, pollIds, tagNames };
}

export default function (data) {
  if (!data || !data.pollIds || data.pollIds.length === 0) return;

  const token = data.accessToken;
  const params = { headers: { 'Content-Type': 'application/json' } };
  const authParams = {
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
  };

  const scenario = randomIntBetween(1, 100);

  if (scenario <= 35) {
    group('GET feed', () => {
      let url = `${BASE_URL}/v1/feed`;
      if (Math.random() < 0.3 && data.tagNames && data.tagNames.length > 0) {
        url += `?tags=${encodeURIComponent(randomItem(data.tagNames))}`;
      }
      const start = Date.now();
      const res = http.get(url, params);
      feedDuration.add(Date.now() - start);
      const ok = check(res, { 'feed 200': (r) => r.status === 200 });
      if (!ok) { errors.add(1); httpErrorsByCode.add(1, { code: String(res.status) }); }
      sleep(0.05);
    });
  } else if (scenario <= 55) {
    group('GET poll', () => {
      const pollId = randomItem(data.pollIds);
      const start = Date.now();
      const res = http.get(`${BASE_URL}/v1/polls/${pollId}`, params);
      pollDuration.add(Date.now() - start);
      const ok = check(res, { 'poll 200': (r) => r.status === 200 });
      if (!ok) { errors.add(1); httpErrorsByCode.add(1, { code: String(res.status) }); }
      sleep(0.03);
    });
  } else if (scenario <= 80) {
    group('POST vote', () => {
      const pollId = randomItem(data.pollIds);
      let res = http.get(`${BASE_URL}/v1/polls/${pollId}`, authParams);
      if (res.status === 200) {
        const poll = res.json().poll;
        if (poll.options && poll.options.length > 0) {
          const count = poll.type === 'POLL_TYPE_MULTIPLE_CHOICE'
            ? randomIntBetween(1, Math.min(3, poll.options.length))
            : 1;
          const shuffled = [...poll.options].sort(() => Math.random() - 0.5);
          const optionIds = shuffled.slice(0, count).map(o => o.id);
          const start = Date.now();
          res = http.post(`${BASE_URL}/v1/polls/${pollId}/vote`, JSON.stringify({
            pollId: pollId, optionIds: optionIds,
          }), authParams);
          voteDuration.add(Date.now() - start);
          const ok = check(res, { 'vote 200': (r) => r.status === 200 });
          if (!ok) { errors.add(1); httpErrorsByCode.add(1, { code: String(res.status) }); }
        }
      }
      sleep(0.05);
    });
  } else {
    group('GET analytics', () => {
      const pollId = randomItem(data.pollIds);
      const endpoints = [
        `${BASE_URL}/v1/polls/${pollId}/analytics`,
        `${BASE_URL}/v1/polls/${pollId}/analytics/countries`,
        `${BASE_URL}/v1/polls/${pollId}/analytics/gender`,
        `${BASE_URL}/v1/polls/${pollId}/analytics/age`,
      ];
      const start = Date.now();
      const res = http.get(randomItem(endpoints), params);
      analyticsDuration.add(Date.now() - start);
      const ok = check(res, { 'analytics 200': (r) => r.status === 200 });
      if (!ok) { errors.add(1); httpErrorsByCode.add(1, { code: String(res.status) }); }
      sleep(0.03);
    });
  }
}

export function teardown(data) {
  if (data && data.accessToken) {
    http.post(`${BASE_URL}/v1/auth/logout-all`, '{}', {
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${data.accessToken}` },
    });
  }
}

export function handleSummary(data) {
  const v = (m) => (m && m.values ? m.values : null);

  const rd = v(data.metrics.http_req_duration);
  const rf = v(data.metrics.http_req_failed);
  const rq = v(data.metrics.http_reqs);
  const fd = v(data.metrics.feed_duration_ms);
  const pd = v(data.metrics.poll_duration_ms);
  const vd = v(data.metrics.vote_duration_ms);
  const ad = v(data.metrics.analytics_duration_ms);
  const er = v(data.metrics.errors);
  const ec = v(data.metrics.http_errors_by_code);
  const ch = v(data.metrics.checks);

  const p = (vals, key) => {
    if (!vals) return '-';
    const got = vals[key];
    return got !== undefined && got !== null ? got.toFixed(1) : '-';
  };

  const summary = {
    timestamp: new Date().toISOString(),
    duration_sec: Math.round(data.state.testRunDurationMs / 1000),
    total_requests: rq ? rq.count : 0,
    avg_rps: rq ? rq.rate.toFixed(1) : '-',
    http_req_failed_pct: rf ? (rf.rate * 100).toFixed(2) : '-',

    latency_ms: {
      avg: rd ? rd.avg.toFixed(1) : '-',
      p50: p(rd, 'p(50)'),
      p95: p(rd, 'p(95)'),
      p99: p(rd, 'p(99)'),
      max: rd ? rd.max.toFixed(1) : '-',
    },

    endpoints_p95_ms: {
      feed:      p(fd, 'p(95)'),
      poll:      p(pd, 'p(95)'),
      vote:      p(vd, 'p(95)'),
      analytics: p(ad, 'p(95)'),
    },

    error_rate_pct: er ? (er.rate * 100).toFixed(2) : '-',
    errors_by_code: ec ? ec.count : {},
    checks_passed: ch ? ch.passes : 0,
    checks_failed: ch ? ch.fails : 0,
  };

  return {
    'stdout': JSON.stringify(summary, null, 2),
    'k6/results.json': JSON.stringify(data.metrics, null, 2),
  };
}
