import { useContext, useEffect, type RefObject } from "react";
import { AuthSuspension } from "./AuthSuspension";
// Native modal focus trapping/inertness plus an explicit, stable return target.
// Closing before removal also works through React StrictMode's effect replay.
export function useModalDialog(
  ref: RefObject<HTMLDialogElement | null>,
  selector = "input,textarea,select,button",
) {
  const suspended = useContext(AuthSuspension);
  useEffect(() => {
    const dialog = ref.current;
    if (!dialog || suspended) return;
    const opener = document.activeElement;
    dialog.showModal();
    dialog.querySelector<HTMLElement>(selector)?.focus();
    return () => {
      dialog.close();
      if (
        opener instanceof HTMLElement &&
        opener.isConnected &&
        !opener.closest("[hidden],[inert]")
      )
        opener.focus();
    };
  }, [ref, selector, suspended]);
}
