function setAzureText(selector, value) {
  const root = document.querySelector(selector);
  if (!root) {
    return;
  }
  root.textContent = String(value);
}

function setupAzureOverview() {
  const catalog = window.AZURE_SERVICE_CATALOG;
  if (!catalog || !catalog.summary) {
    return;
  }

  const summary = catalog.summary;
  setAzureText("[data-azure-provider-services]", summary.providerServices ?? 0);
  setAzureText("[data-azure-services-listed]", summary.servicesListed ?? 0);
  setAzureText("[data-azure-examples]", summary.examplesAvailable ?? 0);
  setAzureText("[data-azure-plans]", summary.plansAvailable ?? 0);
  setAzureText("[data-azure-contract-strict]", summary.contractStrictAllThree ?? 0);
  setAzureText("[data-azure-io-strict]", summary.ioStrictAllFour ?? 0);
}

document.addEventListener("DOMContentLoaded", () => {
  setupAzureOverview();
});
