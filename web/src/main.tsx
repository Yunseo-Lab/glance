import { QueryClientProvider } from "@tanstack/react-query";
import { queryClient } from "./workspace/queries";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./style.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
