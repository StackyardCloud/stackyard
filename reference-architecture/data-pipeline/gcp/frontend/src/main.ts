import "./style.css";

type ApiError = {
  __type?: string;
  message?: string;
};

type TenantState = {
  tenantId: string;
  kmsKeyArn: string;
  kmsKeyId: string;
  secretArn: string;
  queueUrl: string;
  queueArn: string;
  eventBusName: string;
  eventBusArn: string;
  ruleName: string;
  ruleArn: string;
  stateMachineArn: string;
  latestRawKey: string;
  updatedAtUtc: string;
  runs: Array<{
    runId: string;
    executionArn: string;
    rawObjectKey: string;
    curatedObjectKey: string;
    publishedObjectKey: string;
    status: string;
    startedAtUtc: string;
    completedAtUtc: string;
  }>;
};

type SummaryResponse = {
  tenants: TenantState[];
};

type TenantDetails = TenantState & {
  objectsByZone: Record<string, string[]>;
};

const apiBase = (import.meta.env.VITE_API_BASE as string | undefined)?.replace(/\/$/, "") || "/api";

const app = document.querySelector<HTMLDivElement>("#app");
if (!app) {
  throw new Error("Missing #app container");
}

app.innerHTML = `
  <main class="layout">
    <header class="hero">
      <p class="eyebrow">Reference Architecture</p>
      <h1>Data Pipeline Control Plane</h1>
      <p class="subhead">Go backend + TypeScript frontend using Stackyard-emulated GCP services.</p>
    </header>

    <section class="panel actions">
      <label class="field">
        <span>Tenant ID</span>
        <input id="tenant-input" value="tenant-001" pattern="[a-z0-9-]{3,32}" />
      </label>
      <div class="button-row">
        <button data-action="bootstrap">Bootstrap Tenant</button>
        <button data-action="ingest">Ingest Raw Batch</button>
        <button data-action="run">Run Pipeline</button>
        <button data-action="refresh">Refresh</button>
      </div>
      <p id="status" class="status">Ready.</p>
    </section>

    <section class="grid">
      <article class="panel">
        <h2>Tenants</h2>
        <div id="summary"></div>
      </article>

      <article class="panel">
        <h2>Tenant Detail</h2>
        <pre id="detail" class="json">Select or refresh a tenant.</pre>
      </article>
    </section>
  </main>
`;

const tenantInput = document.querySelector<HTMLInputElement>("#tenant-input");
const statusEl = document.querySelector<HTMLParagraphElement>("#status");
const summaryEl = document.querySelector<HTMLDivElement>("#summary");
const detailEl = document.querySelector<HTMLPreElement>("#detail");

if (!tenantInput || !statusEl || !summaryEl || !detailEl) {
  throw new Error("Missing required elements");
}

const setStatus = (message: string, kind: "info" | "ok" | "error" = "info") => {
  statusEl.textContent = message;
  statusEl.dataset.kind = kind;
};

const request = async <T>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(`${apiBase}${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers || {})
    },
    ...init
  });

  const text = await response.text();
  const payload = text ? JSON.parse(text) : {};

  if (!response.ok) {
    const err = payload as ApiError;
    const code = err.__type || "Error";
    const message = err.message || response.statusText;
    throw new Error(`${code}: ${message}`);
  }

  return payload as T;
};

const isResourceNotFound = (error: unknown): boolean => {
  return (error as Error)?.message?.includes("ResourceNotFoundException") ?? false;
};

const normalizeSummary = (raw: unknown): SummaryResponse => {
  const payload = (raw ?? {}) as { tenants?: TenantState[] | null };
  const tenants = Array.isArray(payload.tenants) ? payload.tenants : [];

  return {
    tenants: tenants.map((tenant) => ({
      ...tenant,
      runs: Array.isArray(tenant?.runs) ? tenant.runs : []
    }))
  };
};

const refreshSummary = async (): Promise<SummaryResponse> => {
  const raw = await request<unknown>("/v1/summary");
  const summary = normalizeSummary(raw);
  renderSummary(summary);
  return summary;
};

const refreshTenant = async (tenantId: string): Promise<void> => {
  const tenant = await request<TenantDetails>(`/v1/tenants/${encodeURIComponent(tenantId)}`);
  detailEl.textContent = JSON.stringify(tenant, null, 2);
};

const renderSummary = (summary: SummaryResponse) => {
  const tenants = Array.isArray(summary.tenants) ? summary.tenants : [];
  if (tenants.length === 0) {
    summaryEl.innerHTML = `<p class="muted">No tenants yet.</p>`;
    return;
  }

  const rows = tenants
    .map((tenant) => {
      const runCount = Array.isArray(tenant.runs) ? tenant.runs.length : 0;
      return `
        <tr>
          <td><button class="link-button" data-tenant="${tenant.tenantId}">${tenant.tenantId}</button></td>
          <td>${runCount}</td>
          <td>${tenant.latestRawKey ? "yes" : "no"}</td>
          <td><span class="pill">${tenant.updatedAtUtc || "-"}</span></td>
        </tr>
      `;
    })
    .join("");

  summaryEl.innerHTML = `
    <table>
      <thead>
        <tr><th>Tenant</th><th>Runs</th><th>Raw Data</th><th>Updated</th></tr>
      </thead>
      <tbody>${rows}</tbody>
    </table>
  `;

  for (const button of summaryEl.querySelectorAll<HTMLButtonElement>("button[data-tenant]")) {
    button.addEventListener("click", async () => {
      const tenant = button.dataset.tenant || "";
      tenantInput.value = tenant;
      try {
        setStatus(`Loading ${tenant}...`);
        await refreshTenant(tenant);
        setStatus(`Loaded ${tenant}`, "ok");
      } catch (error) {
        setStatus((error as Error).message, "error");
      }
    });
  }
};

const runAction = async (action: "bootstrap" | "ingest" | "run" | "refresh") => {
  const tenantId = tenantInput.value.trim().toLowerCase();
  if (!/^[a-z0-9-]{3,32}$/.test(tenantId)) {
    setStatus("Tenant ID must match [a-z0-9-]{3,32}", "error");
    return;
  }

  try {
    if (action === "refresh") {
      setStatus("Refreshing...");
      await refreshSummary();
      await refreshTenant(tenantId);
      setStatus("Refresh complete", "ok");
      return;
    }

    setStatus(`${action} started for ${tenantId}...`);
    const actionResponse = await request<{ tenant?: unknown }>(
      `/v1/tenants/${encodeURIComponent(tenantId)}/${action}`,
      { method: "POST" }
    );
    await refreshSummary();

    // Bootstrap returns full tenant state. Render it immediately so the user
    // can proceed even if the immediate follow-up detail fetch is transiently unavailable.
    if (action === "bootstrap" && actionResponse?.tenant) {
      detailEl.textContent = JSON.stringify(actionResponse.tenant, null, 2);
    }

    try {
      await refreshTenant(tenantId);
    } catch (error) {
      if (action === "bootstrap" && isResourceNotFound(error)) {
        setStatus(`bootstrap succeeded for ${tenantId} (detail endpoint not ready yet)`, "ok");
        return;
      }
      throw error;
    }

    setStatus(`${action} succeeded for ${tenantId}`, "ok");
  } catch (error) {
    if (isResourceNotFound(error)) {
      setStatus(
        `tenant ${tenantId} not found in backend memory; run Bootstrap Tenant again`,
        "error"
      );
      return;
    }
    setStatus((error as Error).message, "error");
  }
};

for (const button of document.querySelectorAll<HTMLButtonElement>("button[data-action]")) {
  const action = button.dataset.action as "bootstrap" | "ingest" | "run" | "refresh";
  button.addEventListener("click", () => {
    void runAction(action);
  });
}

void (async () => {
  try {
    await refreshSummary();
    await refreshTenant(tenantInput.value.trim().toLowerCase());
  } catch {
    setStatus("Bootstrap a tenant to begin.");
  }
})();
