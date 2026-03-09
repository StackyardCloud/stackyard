function escapeHtml(text) {
  return String(text)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function setText(selector, value) {
  const root = document.querySelector(selector);
  if (!root) {
    return;
  }
  root.textContent = String(value);
}

function serviceEndpointNodes(service) {
  const nodes = [];
  if (service.docsHref) {
    nodes.push({
      name: "endpoint:docs",
      value: service.docsHref
    });
  }
  if (service.exampleHref) {
    nodes.push({
      name: "endpoint:example",
      value: service.exampleHref
    });
  }
  if (!nodes.length) {
    nodes.push({
      name: "endpoint:pending",
      value: "pending"
    });
  }
  return nodes;
}

function buildTreeData() {
  const providers = [];

  if (
    typeof window !== "undefined" &&
    Array.isArray(window.AWS_SERVICE_CATALOG)
  ) {
    const awsCategories = window.AWS_CATEGORY_CONFIG || {};
    const categories = {};

    window.AWS_SERVICE_CATALOG.forEach((service) => {
      const categoryId = service.category || "uncategorized";
      const categoryName = awsCategories[categoryId]?.label || categoryId;
      if (!categories[categoryId]) {
        categories[categoryId] = {
          name: categoryName,
          services: []
        };
      }
      categories[categoryId].services.push({
        name: service.name,
        id: service.id,
        endpoints: serviceEndpointNodes(service)
      });
    });

    providers.push({
      name: "aws",
      categories: Object.values(categories).sort((a, b) => a.name.localeCompare(b.name))
    });
  }

  if (
    typeof window !== "undefined" &&
    window.GCP_SERVICE_CATALOG &&
    Array.isArray(window.GCP_SERVICE_CATALOG.services)
  ) {
    const gcpCatalog = window.GCP_SERVICE_CATALOG;
    const gcpCategories = gcpCatalog.categories || {};
    const categories = {};

    gcpCatalog.services.forEach((service) => {
      const categoryId = service.category || "uncategorized";
      const categoryName = gcpCategories[categoryId]?.label || categoryId;
      if (!categories[categoryId]) {
        categories[categoryId] = {
          name: categoryName,
          services: []
        };
      }
      categories[categoryId].services.push({
        name: service.name,
        id: service.id,
        endpoints: serviceEndpointNodes(service)
      });
    });

    providers.push({
      name: "gcp",
      categories: Object.values(categories).sort((a, b) => a.name.localeCompare(b.name))
    });
  }

  if (
    typeof window !== "undefined" &&
    window.AZURE_SERVICE_CATALOG &&
    Array.isArray(window.AZURE_SERVICE_CATALOG.services)
  ) {
    const azureCatalog = window.AZURE_SERVICE_CATALOG;
    const azureCategories = azureCatalog.categories || {};
    const categories = {};

    azureCatalog.services.forEach((service) => {
      const categoryId = service.category || "uncategorized";
      const categoryName = azureCategories[categoryId]?.label || categoryId;
      if (!categories[categoryId]) {
        categories[categoryId] = {
          name: categoryName,
          services: []
        };
      }
      categories[categoryId].services.push({
        name: service.name,
        id: service.id,
        endpoints: serviceEndpointNodes(service)
      });
    });

    providers.push({
      name: "azure",
      categories: Object.values(categories).sort((a, b) => a.name.localeCompare(b.name))
    });
  }

  providers.forEach((provider) => {
    provider.categories.forEach((category) => {
      category.services.sort((a, b) => a.name.localeCompare(b.name));
    });
  });

  return {
    name: "root",
    providers: providers.sort((a, b) => a.name.localeCompare(b.name))
  };
}

function renderTree(tree) {
  const providerHtml = tree.providers
    .map((provider) => {
      const categoryHtml = provider.categories
        .map((category) => {
          const serviceHtml = category.services
            .map((service) => {
              const endpointHtml = service.endpoints
                .map(
                  (endpoint) =>
                    `<li><code>${escapeHtml(endpoint.name)}</code>: <code>${escapeHtml(endpoint.value)}</code></li>`
                )
                .join("");

              return `
                <li>
                  <details class="tree-branch">
                    <summary><code>service</code>: ${escapeHtml(service.name)} <code>(${escapeHtml(service.id)})</code></summary>
                    <ul class="tree-list">${endpointHtml}</ul>
                  </details>
                </li>
              `;
            })
            .join("");

          return `
            <li>
              <details class="tree-branch">
                <summary><code>category</code>: ${escapeHtml(category.name)}</summary>
                <ul class="tree-list">${serviceHtml}</ul>
              </details>
            </li>
          `;
        })
        .join("");

      return `
        <li>
          <details class="tree-branch" open>
            <summary><code>provider</code>: ${escapeHtml(provider.name)}</summary>
            <ul class="tree-list">${categoryHtml}</ul>
          </details>
        </li>
      `;
    })
    .join("");

  return `
    <details class="tree-branch" open>
      <summary><code>root</code></summary>
      <ul class="tree-list">${providerHtml}</ul>
    </details>
  `;
}

function renderTreeStats(tree) {
  const providerCount = tree.providers.length;
  const categoryCount = tree.providers.reduce((total, provider) => total + provider.categories.length, 0);
  const serviceCount = tree.providers.reduce(
    (total, provider) => total + provider.categories.reduce((sum, category) => sum + category.services.length, 0),
    0
  );
  const endpointCount = tree.providers.reduce(
    (total, provider) =>
      total +
      provider.categories.reduce(
        (sum, category) => sum + category.services.reduce((acc, service) => acc + service.endpoints.length, 0),
        0
      ),
    0
  );

  setText("[data-tree-providers]", providerCount);
  setText("[data-tree-categories]", categoryCount);
  setText("[data-tree-services]", serviceCount);
  setText("[data-tree-endpoints]", endpointCount);
}

function setupServiceTree() {
  const root = document.querySelector("[data-service-tree]");
  if (!root) {
    return;
  }

  const tree = buildTreeData();
  root.innerHTML = renderTree(tree);
  renderTreeStats(tree);
}

document.addEventListener("DOMContentLoaded", () => {
  setupServiceTree();
});
