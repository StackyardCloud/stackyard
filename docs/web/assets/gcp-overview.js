function setText(selector, value) {
  const root = document.querySelector(selector);
  if (!root) {
    return;
  }
  root.textContent = String(value);
}

function setupGCPOverview() {
  const catalog = window.GCP_SERVICE_CATALOG;
  if (!catalog || !catalog.summary) {
    return;
  }

  const summary = catalog.summary;
  setText("[data-gcp-provider-services]", summary.providerServices ?? 0);
  setText("[data-gcp-services-listed]", summary.servicesListed ?? 0);
  setText("[data-gcp-examples]", summary.examplesAvailable ?? 0);
  setText("[data-gcp-plans]", summary.plansAvailable ?? 0);
  setText("[data-gcp-contract-strict]", summary.contractStrictAllThree ?? 0);
  setText("[data-gcp-io-strict]", summary.ioStrictAllFour ?? 0);
}

document.addEventListener("DOMContentLoaded", () => {
  setupGCPOverview();
});
