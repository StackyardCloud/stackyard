(() => {
  const nav = document.querySelector("[data-service-nav]");
  const filterInput = document.getElementById("service-filter");

  if (!nav) {
    return;
  }

  const links = Array.from(nav.querySelectorAll("a"));

  function setActiveFromHash() {
    if (!window.location.hash) {
      links.forEach((link) => link.classList.remove("is-active"));
      return;
    }

    links.forEach((link) => {
      link.classList.toggle("is-active", link.getAttribute("href") === window.location.hash);
    });
  }

  function filterLinks(query) {
    const normalized = query.trim().toLowerCase();
    links.forEach((link) => {
      const label = (link.textContent || "").toLowerCase();
      link.style.display = !normalized || label.includes(normalized) ? "" : "none";
    });
  }

  if (filterInput) {
    filterInput.addEventListener("input", (event) => {
      filterLinks(event.target.value || "");
    });
  }

  const sections = links
    .map((link) => {
      const targetId = link.getAttribute("href")?.replace(/^#/, "");
      if (!targetId) {
        return null;
      }
      return document.getElementById(targetId);
    })
    .filter(Boolean);

  if (sections.length) {
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];

        if (!visible || !visible.target.id) {
          return;
        }

        const id = `#${visible.target.id}`;
        links.forEach((link) => link.classList.toggle("is-active", link.getAttribute("href") === id));
      },
      {
        rootMargin: "-20% 0px -72% 0px",
        threshold: [0, 1]
      }
    );

    sections.forEach((section) => observer.observe(section));
  }

  window.addEventListener("hashchange", setActiveFromHash);
  setActiveFromHash();
})();
