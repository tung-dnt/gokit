import http from "k6/http";
import { check, sleep } from "k6";
import { Trend, Rate, Counter } from "k6/metrics";

// ── custom metrics ────────────────────────────────────────────────────────────
const listLatency   = new Trend("list_users_latency",   true);
const getLatency    = new Trend("get_user_latency",     true);
const createLatency = new Trend("create_user_latency",  true);
const errorRate     = new Rate("error_rate");
const createdUsers  = new Counter("created_users");

// ── config ────────────────────────────────────────────────────────────────────
const BASE_URL = __ENV.BASE_URL || "http://host.docker.internal:4040";

export const options = {
  scenarios: {
    list_users: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 20 },
        { duration: "30s", target: 50 },
        { duration: "10s", target: 0  },
      ],
      exec: "listUsers",
      tags: { scenario: "list" },
    },
    get_user: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 20 },
        { duration: "30s", target: 50 },
        { duration: "10s", target: 0  },
      ],
      exec: "getUserByID",
      tags: { scenario: "get" },
    },
    create_user: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "10s", target: 5  },
        { duration: "30s", target: 10 },
        { duration: "10s", target: 0  },
      ],
      exec: "createUser",
      tags: { scenario: "create" },
    },
  },
  thresholds: {
    list_users_latency:  ["p(95)<200"],
    get_user_latency:    ["p(95)<100"],
    create_user_latency: ["p(95)<300"],
    error_rate:          ["rate<0.01"],
    http_req_failed:     ["rate<0.01"],
  },
};

// ── per-VU CSRF token ─────────────────────────────────────────────────────────
// Module-level variables are initialised once per VU (each VU has its own JS
// runtime), so this is NOT shared across VUs.
let vuCsrfToken = null;

/**
 * Returns the CSRF token for this VU, fetching it on first call.
 * The _csrf cookie is stored automatically in the VU's cookie jar and will be
 * sent on every subsequent request to the same host.
 */
function csrfToken() {
  if (vuCsrfToken === null) {
    const res = http.get(`${BASE_URL}/api/users`);
    const cookie = res.cookies["_csrf"];
    vuCsrfToken = cookie && cookie.length > 0 ? cookie[0].value : "";
  }
  return vuCsrfToken;
}

// ── seed data (runs once before all scenarios) ────────────────────────────────
export function setup() {
  // setup() runs in its own VU context — get a dedicated token for seeding
  const res = http.get(`${BASE_URL}/api/users`);
  const cookie = res.cookies["_csrf"];
  const token  = cookie && cookie.length > 0 ? cookie[0].value : "";

  const userIDs = [];
  for (let i = 1; i <= 10; i++) {
    const r = http.post(
      `${BASE_URL}/api/users`,
      JSON.stringify({ name: `Seed User ${i}`, email: `seed${i}_${Date.now()}@example.com` }),
      {
        headers: { "Content-Type": "application/json", "X-CSRF-Token": token },
        cookies: { _csrf: token },
      },
    );
    if (r.status === 201) {
      const body = JSON.parse(r.body);
      if (body.id) userIDs.push(body.id);
    }
  }

  return { userIDs };
}

// ── scenario: list users ──────────────────────────────────────────────────────
export function listUsers() {
  const res = http.get(`${BASE_URL}/api/users`);

  listLatency.add(res.timings.duration);
  errorRate.add(res.status !== 200);

  check(res, {
    "list: status 200":    (r) => r.status === 200,
    "list: body is array": (r) => Array.isArray(JSON.parse(r.body)),
  });

  sleep(0.1);
}

// ── scenario: get user by ID ──────────────────────────────────────────────────
export function getUserByID(data) {
  const { userIDs } = data;
  if (!userIDs || userIDs.length === 0) return;

  const id  = userIDs[Math.floor(Math.random() * userIDs.length)];
  const res = http.get(`${BASE_URL}/api/users/${id}`);

  getLatency.add(res.timings.duration);
  errorRate.add(res.status !== 200);

  check(res, {
    "get: status 200":   (r) => r.status === 200,
    "get: has id field": (r) => JSON.parse(r.body).id === id,
  });

  sleep(0.05);
}

// ── scenario: create user ─────────────────────────────────────────────────────
export function createUser() {
  // Each VU fetches and caches its own CSRF token — no sharing across VUs
  const token = csrfToken();
  const uid   = `${__VU}_${__ITER}`;

  const res = http.post(
    `${BASE_URL}/api/users`,
    JSON.stringify({ name: `Load User ${uid}`, email: `load_${uid}@example.com` }),
    {
      headers: { "Content-Type": "application/json", "X-CSRF-Token": token },
      // Cookie is sent automatically from the VU's jar; explicit fallback here
      cookies: { _csrf: token },
    },
  );

  createLatency.add(res.timings.duration);
  errorRate.add(res.status !== 201);

  if (res.status === 201) createdUsers.add(1);

  check(res, {
    "create: status 201": (r) => r.status === 201,
    "create: has id":     (r) => !!JSON.parse(r.body).id,
  });

  sleep(0.2);
}
