const copyButtons = document.querySelectorAll("[data-copy]");

for (const button of copyButtons) {
  button.addEventListener("click", async () => {
    const value = button.getAttribute("data-copy");
    if (!value) return;

    const original = button.textContent;

    try {
      await navigator.clipboard.writeText(value);
      button.textContent = "Copiado";
    } catch {
      button.textContent = "Copia manualmente";
    }

    window.setTimeout(() => {
      button.textContent = original;
    }, 1600);
  });
}
