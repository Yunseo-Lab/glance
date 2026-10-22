import { createContext } from "react";
// A session expiry hides the workspace without destroying in-memory form edits.
// Explicit logout clears and unmounts the workspace separately.
export const AuthSuspension = createContext(false);
