const state = {
  knownKeyIds: [],
  registryItems: [],
  generatedToken: "",
};

document.addEventListener("DOMContentLoaded", async () => {
  bindTabs();
  bindModeToggle();
  bindIssueForm();
  bindVerifyForm();
  bindRegistryFilters();
  bindCopyButtons();
  await refreshBootstrap();
  await loadRegistry();
});

function bindTabs() {
  const tabs = document.querySelectorAll(".tab");
  tabs.forEach((tab) => {
    tab.addEventListener("click", () => {
      document
        .querySelectorAll(".tab")
        .forEach((value) => value.classList.remove("is-active"));
      document
        .querySelectorAll(".panel")
        .forEach((value) => value.classList.remove("is-active"));
      tab.classList.add("is-active");
      const target = document.getElementById(tab.dataset.tabTarget);
      if (target) {
        target.classList.add("is-active");
      }
    });
  });
}

function bindModeToggle() {
  document.querySelectorAll('input[name="license_mode"]').forEach((input) => {
    input.addEventListener("change", syncModeFields);
  });
  syncModeFields();
}

function syncModeFields() {
  const paidSelected = document.querySelector('input[name="license_mode"]:checked')?.value !== "trial";
  document.getElementById("duration-field").classList.toggle("hidden", !paidSelected);
  document.getElementById("trial-field").classList.toggle("hidden", paidSelected);
  document.querySelectorAll(".segment").forEach((segment) => {
    const input = segment.querySelector("input");
    segment.classList.toggle("is-selected", input?.checked === true);
  });
}

function bindIssueForm() {
  const form = document.getElementById("issue-form");
  form.addEventListener("submit", async (event) => {
    event.preventDefault();

    const keyInput = document.getElementById("issue-private-key");
    if (!keyInput.files || keyInput.files.length === 0) {
      showMessage("Select a private key file before generating a license.", true);
      return;
    }

    const data = new FormData();
    data.append("hwid", document.getElementById("issue-hwid").value.trim());
    data.append("kid", document.getElementById("issue-kid").value.trim());
    data.append("tier", document.getElementById("issue-tier").value.trim());
    data.append("orgs", document.getElementById("issue-orgs").value);
    data.append("users", document.getElementById("issue-users").value);
    data.append("wa_endpoints", document.getElementById("issue-wa-endpoints").value);
    data.append("workers", document.getElementById("issue-workers").value);
    data.append("private_key_file", keyInput.files[0]);

    const mode = document.querySelector('input[name="license_mode"]:checked')?.value || "paid";
    if (mode === "trial") {
      data.append("trial", document.getElementById("issue-trial").value);
      data.append("duration", document.getElementById("issue-duration").value);
    } else {
      data.append("duration", document.getElementById("issue-duration").value);
    }

    try {
      const response = await fetch("/api/issue", {
        method: "POST",
        body: data,
      });
      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload.error || "Failed to generate license");
      }

      state.generatedToken = payload.token;
      document.getElementById("generated-token").value = payload.token;
      document.getElementById("meta-license-id").textContent = payload.entry.license_id;
      document.getElementById("meta-family-id").textContent = payload.entry.license_family_id;
      document.getElementById("meta-kind").textContent = `${payload.entry.license_kind} • ${payload.entry.duration_preset}`;
      document.getElementById("meta-expiry").textContent = payload.entry.expires_at || "Lifetime";

      updateSummary(payload.summary);
      state.knownKeyIds = payload.known_key_ids || [];
      renderKnownKids();
      showMessage("Security key generated and saved to the local registry.");
      keyInput.value = "";
      await loadRegistry();
    } catch (error) {
      showMessage(error.message || "Failed to generate token", true);
    }
  });
}

function bindVerifyForm() {
  document.getElementById("verify-button").addEventListener("click", async () => {
    const token = document.getElementById("verify-token").value.trim();
    if (!token) {
      showMessage("Paste a token before verifying.", true);
      return;
    }

    try {
      const response = await fetch("/api/verify", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token }),
      });
      const payload = await response.json();
      renderVerifyResult(payload);
    } catch (error) {
      showMessage(error.message || "Verify request failed", true);
    }
  });
}

function bindRegistryFilters() {
  ["filter-hwid", "filter-tier", "filter-kind", "filter-status"].forEach((id) => {
    document.getElementById(id).addEventListener("input", () => loadRegistry());
    document.getElementById(id).addEventListener("change", () => loadRegistry());
  });
  document.getElementById("refresh-registry-button").addEventListener("click", () => loadRegistry());
}

function bindCopyButtons() {
  document.getElementById("copy-token-button").addEventListener("click", async () => {
    if (!state.generatedToken) {
      showMessage("Generate a token first.", true);
      return;
    }
    await navigator.clipboard.writeText(state.generatedToken);
    showMessage("Generated security key copied to clipboard.");
  });
}

async function refreshBootstrap() {
  const response = await fetch("/api/bootstrap");
  const payload = await response.json();
  if (!response.ok) {
    showMessage(payload.error || "Failed to load bootstrap data", true);
    return;
  }

  state.knownKeyIds = payload.known_key_ids || [];
  renderKnownKids();
  updateSummary(payload.summary);

  document.getElementById("registry-path").textContent = payload.registry_path;
  document.getElementById("keyring-path").textContent = payload.keyring_path;
  document.getElementById("issue-kid").value = payload.defaults.kid;
  document.getElementById("issue-tier").value = payload.defaults.tier;
  document.getElementById("issue-duration").value = payload.defaults.duration;
  document.getElementById("issue-orgs").value = payload.defaults.orgs;
  document.getElementById("issue-users").value = payload.defaults.users;
  document.getElementById("issue-wa-endpoints").value = payload.defaults.wa_endpoints;
  document.getElementById("issue-workers").value = payload.defaults.workers;
}

function renderKnownKids() {
  const datalist = document.getElementById("known-kids");
  datalist.innerHTML = "";
  state.knownKeyIds.forEach((kid) => {
    const option = document.createElement("option");
    option.value = kid;
    datalist.appendChild(option);
  });
  document.getElementById("known-kids-pill").textContent =
    state.knownKeyIds.length > 0 ? `Known KIDs: ${state.knownKeyIds.join(", ")}` : "Known KIDs: none";
}

function updateSummary(summary) {
  document.getElementById("summary-total").textContent = summary.total ?? 0;
  document.getElementById("summary-trials").textContent = summary.trials ?? 0;
  document.getElementById("summary-paid").textContent = summary.paid ?? 0;
  document.getElementById("summary-active").textContent = summary.active ?? 0;
  document.getElementById("summary-expired").textContent = summary.expired ?? 0;
  document.getElementById("summary-lifetime").textContent = summary.lifetime ?? 0;
}

async function loadRegistry() {
  const params = new URLSearchParams();
  const hwid = document.getElementById("filter-hwid").value.trim();
  const tier = document.getElementById("filter-tier").value.trim();
  const kind = document.getElementById("filter-kind").value;
  const status = document.getElementById("filter-status").value;
  if (hwid) params.set("hwid", hwid);
  if (tier) params.set("tier", tier);
  if (kind) params.set("kind", kind);
  if (status) params.set("status", status);

  const response = await fetch(`/api/licenses?${params.toString()}`);
  const payload = await response.json();
  if (!response.ok) {
    showMessage(payload.error || "Failed to load registry", true);
    return;
  }
  state.registryItems = payload.items || [];
  renderRegistry();
}

function renderRegistry() {
  const body = document.getElementById("registry-body");
  body.innerHTML = "";

  if (state.registryItems.length === 0) {
    body.innerHTML = '<tr><td colspan="9" class="empty-state">No licenses match the current filters.</td></tr>';
    return;
  }

  state.registryItems.forEach((item) => {
    const row = document.createElement("tr");
    row.innerHTML = `
      <td><strong>${escapeHtml(shorten(item.hwid, 16))}</strong><br /><small>${escapeHtml(item.license_id)}</small></td>
      <td>${escapeHtml(item.tier)}</td>
      <td>${escapeHtml(item.license_kind)}</td>
      <td>${escapeHtml(item.duration_preset || "lifetime")}</td>
      <td>${item.orgs}/${item.users}/${item.wa_endpoints}/${item.workers}</td>
      <td>${formatDate(item.issued_at)}</td>
      <td>${item.expires_at ? formatDate(item.expires_at) : "Lifetime"}</td>
      <td><span class="status-badge ${escapeHtml(item.status)}">${escapeHtml(item.status)}</span></td>
      <td><button class="secondary-button token-copy-button" data-license-id="${escapeHtml(item.id)}" type="button">Copy</button></td>
    `;
    body.appendChild(row);
  });

  document.querySelectorAll(".token-copy-button").forEach((button) => {
    button.addEventListener("click", async () => {
      const response = await fetch(`/api/licenses/${button.dataset.licenseId}/token`);
      const payload = await response.json();
      if (!response.ok) {
        showMessage(payload.error || "Failed to load token", true);
        return;
      }
      await navigator.clipboard.writeText(payload.token);
      showMessage("Stored security key copied to clipboard.");
    });
  });
}

function renderVerifyResult(result) {
  const title = document.getElementById("verify-status-title");
  const pill = document.getElementById("verify-status-pill");
  const message = document.getElementById("verify-message");
  const json = document.getElementById("verify-json");

  let pillClass = "verify-pill-invalid";
  if (result.status === "valid_tracked") {
    title.textContent = "Valid + tracked";
    pillClass = "verify-pill-tracked";
  } else if (result.status === "valid_untracked") {
    title.textContent = "Valid + untracked";
    pillClass = "verify-pill-untracked";
  } else {
    title.textContent = "Invalid";
  }

  pill.className = `pill ${pillClass}`;
  pill.textContent = result.status;
  message.textContent = result.message || "No message";
  json.textContent = JSON.stringify(result, null, 2);
}

function showMessage(text, isError = false) {
  const node = document.getElementById("app-message");
  node.textContent = text;
  node.hidden = false;
  node.classList.toggle("is-error", isError);
}

function formatDate(value) {
  if (!value) return "-";
  return new Date(value).toLocaleString();
}

function shorten(value, size) {
  if (!value || value.length <= size) return value || "-";
  return `${value.slice(0, size)}…`;
}

function escapeHtml(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
