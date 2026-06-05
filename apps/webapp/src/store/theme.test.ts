import { beforeEach, describe, expect, it } from "vitest";
import { applyTheme, initialTheme, useTheme } from "./theme";

describe("themeStore", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    useTheme.setState({ mode: "dark" });
  });

  it("initialTheme padrão é dark", () => {
    expect(initialTheme()).toBe("dark");
  });

  it("initialTheme respeita o valor salvo", () => {
    localStorage.setItem("theme", "light");
    expect(initialTheme()).toBe("light");
  });

  it("applyTheme alterna a classe dark no html", () => {
    applyTheme("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    applyTheme("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("toggle alterna e persiste", () => {
    useTheme.getState().set("light");
    expect(localStorage.getItem("theme")).toBe("light");
    useTheme.getState().toggle();
    expect(useTheme.getState().mode).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });
});
