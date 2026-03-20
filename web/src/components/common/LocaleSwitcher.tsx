import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useI18n } from "../../i18n/context";

export default function LocaleSwitcher() {
  const { locale, setLocale, tr } = useI18n();
  const [localeOpen, setLocaleOpen] = useState(false);
  const localeTriggerRef = useRef<HTMLButtonElement | null>(null);
  const localeMenuRef = useRef<HTMLDivElement | null>(null);
  const [localeMenuPos, setLocaleMenuPos] = useState<{ top: number; right: number } | null>(null);

  const updatePos = useCallback(() => {
    if (typeof window === "undefined") return;
    const btn = localeTriggerRef.current;
    if (!btn) return;
    const rect = btn.getBoundingClientRect();
    setLocaleMenuPos({
      top: rect.bottom + 8,
      right: window.innerWidth - rect.right,
    });
  }, []);

  useLayoutEffect(() => {
    if (!localeOpen) return;
    updatePos();
  }, [localeOpen, updatePos]);

  useEffect(() => {
    if (!localeOpen) return;

    const onResize = () => updatePos();
    const onScroll = () => updatePos();
    window.addEventListener("resize", onResize);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      window.removeEventListener("resize", onResize);
      window.removeEventListener("scroll", onScroll, true);
    };
  }, [localeOpen, updatePos]);

  useEffect(() => {
    if (!localeOpen) return;

    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node | null;
      if (!target) return;
      if (localeTriggerRef.current?.contains(target)) return;
      if (localeMenuRef.current?.contains(target)) return;
      setLocaleOpen(false);
    };

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setLocaleOpen(false);
    };

    document.addEventListener("pointerdown", onPointerDown);
    window.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [localeOpen]);

  return (
    <>
      <button
        ref={localeTriggerRef}
        type="button"
        className={`bp-locale-trigger${localeOpen ? " bp-locale-trigger-open" : ""}`}
        aria-label={tr("nav.language", "Language")}
        title={tr("nav.language", "Language")}
        aria-haspopup="menu"
        aria-expanded={localeOpen}
        onMouseDown={(e) => {
          // Prevent the opening gesture from being treated as "outside click" by other handlers.
          e.stopPropagation();
        }}
        onClick={(e) => {
          e.stopPropagation();
          setLocaleOpen((open) => {
            const next = !open;
            if (next) updatePos();
            return next;
          });
        }}
      >
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Zm6.94 9h-3.03a15.3 15.3 0 0 0-1.2-5.03A8.02 8.02 0 0 1 18.94 11Zm-6.94-7a13.3 13.3 0 0 1 1.95 7h-3.9A13.3 13.3 0 0 1 12 4ZM4.06 13h3.03c.14 1.78.55 3.51 1.2 5.03A8.02 8.02 0 0 1 4.06 13Zm3.03-2H4.06a8.02 8.02 0 0 1 4.23-5.03A15.3 15.3 0 0 0 7.09 11Zm4.91 9a13.3 13.3 0 0 1-1.95-7h3.9A13.3 13.3 0 0 1 12 20Zm.71-2.97A15.3 15.3 0 0 0 15.91 13h3.03a8.02 8.02 0 0 1-4.23 4.03Z" />
        </svg>
        <span>{locale.toUpperCase()}</span>
      </button>

      {localeOpen && typeof document !== "undefined"
        ? createPortal(
            <div
              ref={localeMenuRef}
              className="bp-locale-menu bp-locale-menu-pop"
              style={
                localeMenuPos ? { top: localeMenuPos.top, right: localeMenuPos.right } : undefined
              }
              role="menu"
              aria-label={tr("nav.language", "Language")}
            >
              <button
                type="button"
                className={
                  locale === "zh" ? "bp-locale-option bp-locale-option-active" : "bp-locale-option"
                }
                onClick={() => {
                  setLocale("zh");
                  setLocaleOpen(false);
                }}
              >
                <span>{tr("nav.language.zh", "中文")}</span>
                {locale === "zh" ? <span>✓</span> : null}
              </button>
              <button
                type="button"
                className={
                  locale === "en" ? "bp-locale-option bp-locale-option-active" : "bp-locale-option"
                }
                onClick={() => {
                  setLocale("en");
                  setLocaleOpen(false);
                }}
              >
                <span>{tr("nav.language.en", "English")}</span>
                {locale === "en" ? <span>✓</span> : null}
              </button>
            </div>,
            document.body,
          )
        : null}
    </>
  );
}
