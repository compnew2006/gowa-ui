(function () {
  var mode = localStorage.getItem("color-mode");
  var isDark =
    mode === "dark" ||
    ((mode === "system" || !mode) &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);
  if (isDark) {
    document.documentElement.classList.add("dark");
  }
})();
