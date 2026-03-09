function compareAzureServices(a, b) {
  return a.name.localeCompare(b.name);
}

function normalizeAzureText(text) {
  return text.toLowerCase().trim();
}

function azureCategoryCount(services, category) {
  if (category === "all") {
    return services.length;
  }
  return services.filter((service) => service.category === category).length;
}

function azureMatchesQuery(service, query, categories) {
  if (!query) {
    return true;
  }

  const category = categories[service.category];
  const haystack = [
    service.name,
    service.summary,
    service.id,
    service.canonicalService,
    category ? category.label : ""
  ]
    .join(" ")
    .toLowerCase();

  return haystack.includes(query);
}

function renderAzureSidebarNav(services) {
  const navRoot = document.querySelector("[data-service-nav]");
  if (!navRoot) {
    return;
  }

  const links = [...services]
    .sort(compareAzureServices)
    .map(
      (service) =>
        `<a href="#service-${service.id}" title="Jump to ${service.name}">${service.name}</a>`
    )
    .join("");

  navRoot.innerHTML = links;
}

function renderAzureFilterChips(services, activeCategory, categories, categoryOrder) {
  const chipRoot = document.querySelector("[data-filter-chips]");
  if (!chipRoot) {
    return;
  }

  const allCategories = ["all", ...categoryOrder];
  chipRoot.innerHTML = allCategories
    .map((category) => {
      const config = categories[category];
      if (!config) {
        return "";
      }
      const count = azureCategoryCount(services, category);
      const activeClass = category === activeCategory ? "active" : "";
      return `<button class="filter-chip ${activeClass}" type="button" data-category="${category}">${config.label} (${count})</button>`;
    })
    .join("");
}

function renderAzureCatalog(services, state, categories, categoryOrder) {
  const catalogRoot = document.querySelector("[data-services]");
  const countRoot = document.querySelector("[data-visible-count]");

  if (!catalogRoot) {
    return;
  }

  const filtered = services
    .filter((service) => azureMatchesQuery(service, state.query, categories))
    .filter((service) => state.category === "all" || service.category === state.category)
    .sort(compareAzureServices);

  if (countRoot) {
    countRoot.textContent = `Showing ${filtered.length} of ${services.length} services`;
  }

  if (!filtered.length) {
    catalogRoot.innerHTML = `
      <div class="empty-state reveal">
        <strong>No matching services</strong>
        <p>Try another keyword or switch to a different category filter.</p>
      </div>
    `;
    return;
  }

  const categoriesToRender = state.category === "all" ? categoryOrder : [state.category];
  const content = categoriesToRender
    .map((category) => {
      const categoryServices = filtered.filter((service) => service.category === category);
      if (!categoryServices.length) {
        return "";
      }

      const config = categories[category];
      if (!config) {
        return "";
      }

      const cards = categoryServices
        .map((service) => {
          const contractScore = service.capabilities?.contractScore;
          const ioScore = service.capabilities?.ioScore;
          const fullCoverage = contractScore === 3 && ioScore === 4;

          const statusPill = fullCoverage
            ? `<span class="pill warn">Contract Ready</span>`
            : `<span class="pill warn">Coverage In Progress</span>`;

          const capabilityTags = [
            contractScore !== null && contractScore !== undefined
              ? `<span class="pill neutral">Contract ${contractScore}/3</span>`
              : `<span class="pill neutral">Contract N/A</span>`,
            ioScore !== null && ioScore !== undefined
              ? `<span class="pill neutral">I/O ${ioScore}/4</span>`
              : `<span class="pill neutral">I/O N/A</span>`,
            service.docsHref
              ? `<span class="pill accent">Staged plan linked</span>`
              : `<span class="pill neutral">Plan pending</span>`
          ].join("");

          const docsAction = service.docsHref
            ? `<a class="btn primary" href="${service.docsHref}">Implementation Plan</a>`
            : `<span class="btn disabled">Plan Pending</span>`;

          const exampleAction = service.exampleHref
            ? `<a class="btn ghost" href="${service.exampleHref}">SDK Example</a>`
            : `<span class="btn disabled">No Example Yet</span>`;

          return `
            <article class="service-card" id="service-${service.id}">
              <div class="service-head">
                <h3>${service.name}</h3>
                ${statusPill}
              </div>
              <p class="service-summary">${service.summary}</p>
              <p class="service-summary"><code>${service.id}</code></p>
              <div class="service-tags">${capabilityTags}</div>
              <div class="service-actions">${docsAction}${exampleAction}</div>
            </article>
          `;
        })
        .join("");

      return `
        <section class="catalog-group reveal" id="category-${category}">
          <header class="catalog-header">
            <div>
              <h2>${config.label}</h2>
              <p>${config.description}</p>
            </div>
            <span class="pill neutral">${categoryServices.length} services</span>
          </header>
          <div class="card-grid">${cards}</div>
        </section>
      `;
    })
    .join("");

  catalogRoot.innerHTML = content;
}

function renderAzureStats(catalog) {
  const services = catalog.services;

  const totalRoot = document.querySelector("[data-total-services]");
  const examplesRoot = document.querySelector("[data-example-services]");
  const docsRoot = document.querySelector("[data-documented-services]");
  const fullRoot = document.querySelector("[data-full-gates]");

  if (totalRoot) {
    totalRoot.textContent = String(services.length);
  }
  if (examplesRoot) {
    examplesRoot.textContent = String(services.filter((service) => Boolean(service.exampleHref)).length);
  }
  if (docsRoot) {
    docsRoot.textContent = String(services.filter((service) => Boolean(service.docsHref)).length);
  }
  if (fullRoot) {
    fullRoot.textContent = String(catalog.summary?.contractStrictAllThree ?? 0);
  }
}

function setupAzureCatalog() {
  const catalogRoot = document.querySelector("[data-services]");
  const catalog = window.AZURE_SERVICE_CATALOG;
  if (!catalogRoot || !catalog || !Array.isArray(catalog.services)) {
    return;
  }

  const services = catalog.services;
  const categories = catalog.categories || {};
  const categoryOrder = Array.isArray(catalog.categoryOrder) ? catalog.categoryOrder : [];

  const searchInput = document.querySelector("[data-service-search]");
  const chipRoot = document.querySelector("[data-filter-chips]");

  const state = {
    query: "",
    category: "all"
  };

  renderAzureStats(catalog);
  renderAzureSidebarNav(services);
  renderAzureFilterChips(services, state.category, categories, categoryOrder);
  renderAzureCatalog(services, state, categories, categoryOrder);

  if (searchInput) {
    searchInput.addEventListener("input", (event) => {
      state.query = normalizeAzureText(event.target.value);
      renderAzureCatalog(services, state, categories, categoryOrder);
    });
  }

  if (chipRoot) {
    chipRoot.addEventListener("click", (event) => {
      const target = event.target;
      if (!(target instanceof HTMLElement)) {
        return;
      }
      const category = target.dataset.category;
      if (!category) {
        return;
      }

      state.category = category;
      renderAzureFilterChips(services, state.category, categories, categoryOrder);
      renderAzureCatalog(services, state, categories, categoryOrder);
    });
  }
}

document.addEventListener("DOMContentLoaded", () => {
  setupAzureCatalog();
});
