(() => {
  const navRoot = document.querySelector("[data-service-nav]");
  const filterInput = document.getElementById("service-filter");
  const sectionsRoot = document.getElementById("gcp-service-sections");

  function escapeHtml(text) {
    return String(text)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }

  function setText(selector, value) {
    const node = document.querySelector(selector);
    if (!node) {
      return;
    }
    node.textContent = String(value);
  }

  function categoryLabel(catalog, service) {
    const category = catalog.categories?.[service.category];
    return category ? category.label : service.category;
  }

  function planHrefForLegacy(service) {
    if (!service.docsHref) {
      return null;
    }
    const parts = service.docsHref.split("/");
    return parts[parts.length - 1] || null;
  }

  function exampleHrefForLegacy(service) {
    return `../examples/gcp/${service.id}/docker-compose.yml`;
  }

  function composeCommand(service) {
    return `docker compose -f examples/gcp/${service.id}/docker-compose.yml up --build`;
  }

  function capabilityStatus(value) {
    return value ? "Yes" : "No";
  }

  function renderServiceSection(catalog, service) {
    const safeId = escapeHtml(service.id);
    const safeName = escapeHtml(service.name);
    const safeSummary = escapeHtml(service.summary);
    const safeCategory = escapeHtml(categoryLabel(catalog, service));
    const safeCanonical = escapeHtml(service.canonicalService || service.id);

    const contractScore = service.capabilities?.contractScore;
    const ioScore = service.capabilities?.ioScore;

    const planHref = planHrefForLegacy(service);
    const planItem = planHref
      ? `<li>Staged plan: <a href="${escapeHtml(planHref)}"><code>${escapeHtml(planHref)}</code></a></li>`
      : "<li>Staged plan: pending</li>";

    const exampleHref = exampleHrefForLegacy(service);

    const rows = [
      ["Request validation", capabilityStatus(Boolean(service.capabilities?.requestValidation))],
      ["Typed success fixtures", capabilityStatus(Boolean(service.capabilities?.typedFixtures))],
      ["Negative contract tests", capabilityStatus(Boolean(service.capabilities?.negativeTests))],
      ["I/O validation implementation", capabilityStatus(Boolean(service.capabilities?.ioValidationImpl))],
      ["I/O validation tests", capabilityStatus(Boolean(service.capabilities?.ioValidationTests))],
      ["I/O shape tests", capabilityStatus(Boolean(service.capabilities?.ioShapeTests))]
    ]
      .map(
        ([name, status]) =>
          `<tr><td>${escapeHtml(name)}</td><td><code>${escapeHtml(status)}</code></td></tr>`
      )
      .join("");

    return `
      <section id="${safeId}" class="service">
        <div class="service-header">
          <h2>${safeName}</h2>
          <p>${safeSummary}</p>
        </div>

        <div class="service-meta">
          <span class="meta-pill">Service ID: ${safeId}</span>
          <span class="meta-pill">Canonical: ${safeCanonical}</span>
          <span class="meta-pill">Category: ${safeCategory}</span>
          <span class="meta-pill">Contract score: ${escapeHtml(contractScore ?? "N/A")}/3</span>
          <span class="meta-pill">I/O score: ${escapeHtml(ioScore ?? "N/A")}/4</span>
        </div>

        <details class="details">
          <summary>Capability Gates</summary>
          <table class="table">
            <thead>
              <tr>
                <th>Capability</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </details>

        <details class="details">
          <summary>References</summary>
          <ul>
            ${planItem}
            <li>Example compose file: <a href="${escapeHtml(exampleHref)}"><code>${escapeHtml(exampleHref)}</code></a></li>
            <li>Run example: <code>${escapeHtml(composeCommand(service))}</code></li>
          </ul>
        </details>
      </section>
    `;
  }

  function buildLinks(services) {
    return services
      .map((service) => `<a href="#${escapeHtml(service.id)}">${escapeHtml(service.name)}</a>`)
      .join("");
  }

  function setActiveLink(links) {
    const currentHash = window.location.hash;
    links.forEach((link) => {
      link.classList.toggle("is-active", currentHash && link.getAttribute("href") === currentHash);
    });
  }

  function wireSearch(links) {
    if (!filterInput) {
      return;
    }

    filterInput.addEventListener("input", (event) => {
      const query = String(event.target.value || "").trim().toLowerCase();
      links.forEach((link) => {
        const text = (link.textContent || "").toLowerCase();
        link.style.display = !query || text.includes(query) ? "" : "none";
      });
    });
  }

  function wireIntersection(links) {
    const sections = links
      .map((link) => link.getAttribute("href")?.replace(/^#/, ""))
      .filter(Boolean)
      .map((id) => document.getElementById(id))
      .filter(Boolean);

    if (!sections.length) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];

        if (!visible || !visible.target.id) {
          return;
        }

        const targetHash = `#${visible.target.id}`;
        links.forEach((link) => {
          link.classList.toggle("is-active", link.getAttribute("href") === targetHash);
        });
      },
      {
        rootMargin: "-20% 0px -72% 0px",
        threshold: [0, 1]
      }
    );

    sections.forEach((section) => observer.observe(section));
  }

  function render() {
    const catalog = window.GCP_SERVICE_CATALOG;
    if (!catalog || !Array.isArray(catalog.services) || !navRoot || !sectionsRoot) {
      return;
    }

    const services = [...catalog.services].sort((a, b) => a.name.localeCompare(b.name));

    navRoot.innerHTML = buildLinks(services);
    sectionsRoot.innerHTML = services.map((service) => renderServiceSection(catalog, service)).join("\n");

    setText("[data-provider-services]", catalog.summary?.providerServices ?? 0);
    setText("[data-services-listed]", catalog.summary?.servicesListed ?? services.length);
    setText("[data-plan-count]", catalog.summary?.plansAvailable ?? 0);
    setText("[data-example-count]", catalog.summary?.examplesAvailable ?? 0);
    setText("[data-contract-strict]", catalog.summary?.contractStrictAllThree ?? 0);
    setText("[data-io-strict]", catalog.summary?.ioStrictAllFour ?? 0);

    const links = Array.from(navRoot.querySelectorAll("a"));
    wireSearch(links);
    wireIntersection(links);

    window.addEventListener("hashchange", () => setActiveLink(links));
    setActiveLink(links);
  }

  document.addEventListener("DOMContentLoaded", render);
})();
