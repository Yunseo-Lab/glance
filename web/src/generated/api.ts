// Generated from docs/dashboard-api.openapi.yaml; do not edit.
export interface paths {
    "/api/v1/connections": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["listConnections"];
        put?: never;
        post: operations["createConnection"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/connections/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get: operations["getConnection"];
        put?: never;
        post?: never;
        delete: operations["deleteConnection"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/connections/{id}/test": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: operations["testConnection"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/connections/{id}/revoke": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: operations["revokeConnection"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/sources/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        post?: never;
        delete: operations["deleteSource"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/boards/{id}/validate": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: operations["validateBoard"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/boards/{id}/snapshots": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get: operations["getBoardSnapshots"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/sources": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["getSourceStatuses"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/diagnostics": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["getDiagnostics"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/sources/{id}/refresh": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: operations["requestSourceRefresh"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/sources/{id}/pause": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: operations["pauseSource"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/limits": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["getCapacityLimits"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/boards": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["listBoards"];
        put?: never;
        post: operations["createBoard"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/boards/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        get: operations["getBoard"];
        put: operations["updateBoard"];
        post?: never;
        delete: operations["deleteBoard"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/catalog": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["getCatalog"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/session": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["getOwnerSession"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/login": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: operations["loginOwner"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/logout": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        post: operations["logoutOwner"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/readyz": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["readiness"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/healthz": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["health"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/dashboard": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["getDashboard"];
        put: operations["saveDashboard"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/dashboard/validate": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** @description Validate an import preview without saving, fetching data, or advancing revision. Requires a same-origin Origin header. */
        post: operations["validateDashboard"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/snapshots": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get: operations["getSnapshots"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        Error: {
            /** @description Stable HTTP error class such as unauthorized or conflict */
            code: string;
            error: string;
        };
        Board: {
            /** @description Stable board identifier; home is the backward-compatible default. */
            id: string;
            title: string;
            /** Format: int64 */
            revision: number;
            /** @enum {string} */
            theme: "light" | "dark";
            widgets: components["schemas"]["Widget"][];
            sources: components["schemas"]["Source"][];
        };
        Widget: {
            /** @description Omitted in original v1 documents; omission means version 1. Unknown versions are rejected. */
            configVersion?: number;
            id: string;
            /** @enum {string} */
            type: "clock" | "calendar" | "note" | "bookmarks" | "rss" | "weather";
            title: string;
            /** @enum {string} */
            size: "small" | "wide";
            /** @description Required for RSS/weather; must reference a source of the same type */
            sourceId?: string;
            config: components["schemas"]["WidgetConfig"];
        };
        Source: {
            /** @description Omitted in original v1 documents; omission means version 1. Unknown versions are rejected. */
            configVersion?: number;
            id: string;
            /** @enum {string} */
            type: "rss" | "weather";
            config: components["schemas"]["SourceConfig"];
        };
        Link: {
            title: string;
            url: string;
        };
        /** @description Per-type fields are strictly validated using the server catalog. */
        WidgetConfig: {
            timezone?: string;
            body?: string;
            links?: components["schemas"]["Link"][];
            limit?: number;
        };
        SourceConfig: {
            url?: string;
            latitude?: number;
            longitude?: number;
            location?: string;
        };
        Snapshot: {
            /** @description Safe provider error class; contains no remote body, URL or credential */
            errorCode?: string;
            /** Format: date-time */
            attemptedAt?: string;
            /** @enum {string} */
            state: "loading" | "fresh" | "stale" | "error" | "paused";
            /** Format: date-time */
            updatedAt?: string;
            /** Format: date-time */
            expiresAt?: string;
            /** Format: date-time */
            retryAt?: string;
            error?: string;
            data?: components["schemas"]["FeedData"] | components["schemas"]["WeatherData"];
        };
        FeedData: {
            title: string;
            items: {
                title: string;
                /** Format: uri */
                url: string;
                /** Format: date-time */
                publishedAt?: string;
            }[];
        };
        WeatherData: {
            location: string;
            temperature: number;
            feelsLike: number;
            humidity: number;
            code: number;
        };
        Connection: {
            id: string;
            /** @enum {string} */
            kind: "github" | "calendar" | "json";
            name: string;
            /** @description Public origin only; private paths and query tokens are encrypted */
            endpoint: string;
            hasSecret: boolean;
            generation: number;
            status: string;
            /** Format: date-time */
            createdAt: string;
            /** Format: date-time */
            lastTestedAt?: string;
        };
        ConnectionInput: {
            name: string;
            /** @enum {string} */
            kind: "github" | "calendar" | "json";
            /** @description Required for calendar/json; stored encrypted. GitHub uses api.github.com. */
            endpoint?: string;
            /** Format: password */
            token?: string;
        };
        ConnectionTestReport: {
            connected: boolean;
            identity?: string;
            capabilities: string[];
            errorCode?: string;
        };
        SourceStatus: {
            source: components["schemas"]["Source"];
            boardIds: string[];
            paused: boolean;
            generation: number;
            snapshot: components["schemas"]["Snapshot"];
            running: boolean;
            /** Format: date-time */
            nextRunAt?: string;
            /** Format: date-time */
            nextManualAt?: string;
        };
    };
    responses: {
        /** @description Invalid dashboard, widget or source */
        ValidationError: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/json": components["schemas"]["Error"];
            };
        };
    };
    parameters: never;
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    listConnections: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Safe owner connection metadata; never includes ciphertext, tokens or private calendar URLs */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Connection"][];
                };
            };
        };
    };
    createConnection: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ConnectionInput"];
            };
        };
        responses: {
            /** @description Encrypted connection created */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Connection"];
                };
            };
            /** @description Original encryption key unavailable; never replaced for existing ciphertext */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    getConnection: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Safe connection metadata */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Connection"];
                };
            };
        };
    };
    deleteConnection: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Unreferenced owner connection removed */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Remove referring sources first */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    testConnection: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Endpoint-tested capability, not a claim of access to every private resource */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ConnectionTestReport"];
                };
            };
            /** @description Connection test cooldown; Retry-After header */
            429: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    revokeConnection: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Secret cleared, referring sources paused and old generations invalidated */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    deleteSource: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Unreferenced source removed; saved reading state preserved */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Remove board references first */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    validateBoard: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["Board"];
            };
        };
        responses: {
            /** @description Preview-only validation; no persistence or fetch */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    getBoardSnapshots: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Snapshot map for the owner board */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    getSourceStatuses: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Owner sources with board references, safe snapshot metadata, pause and job states; excludes snapshot body */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["SourceStatus"][];
                };
            };
        };
    };
    getDiagnostics: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Source/job counts and capacity limits without private payloads or credentials */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    requestSourceRefresh: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Accepted within owner and source quota; includes nextAllowedAt */
            202: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Cooldown; Retry-After header explains when to retry. Does not bypass provider backoff. */
            429: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    pauseSource: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": {
                    paused: boolean;
                };
            };
        };
        responses: {
            /** @description Source pause/resume applied, old refresh generation invalidated */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    getCapacityLimits: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Owner instance capacity policy */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        [key: string]: number;
                    };
                };
            };
        };
    };
    listBoards: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Owner-scoped boards, default board first */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Board"][];
                };
            };
        };
    };
    createBoard: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": {
                    title: string;
                    cloneFrom?: string;
                };
            };
        };
        responses: {
            /** @description New board with revision 1; at most 10 owner boards */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Board"];
                };
            };
        };
    };
    getBoard: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Owner board */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Board"];
                };
            };
        };
    };
    updateBoard: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["Board"];
            };
        };
        responses: {
            /** @description Atomically saved board; ID must match the route and revision must be current */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Revision conflict */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    deleteBoard: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": {
                    revision: number;
                };
            };
        };
        responses: {
            /** @description Board deleted; home is protected */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Revision conflict */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    getCatalog: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Versioned, build-time widget and provider catalog; never includes credentials */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        version: number;
                        widgets: Record<string, never>[];
                        providers: Record<string, never>[];
                    };
                };
            };
        };
    };
    getOwnerSession: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Authentication enabled/authenticated booleans; no private dashboard data */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    loginOwner: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": {
                    /** Format: password */
                    password: string;
                };
            };
        };
        responses: {
            /** @description Signed, HttpOnly, SameSite=Strict session set; 12 hour expiry */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Invalid password */
            401: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Login attempt limit exceeded; retry in 60 seconds */
            429: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    logoutOwner: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Current session revoked and cookie cleared */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    readiness: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Database reachable and embedded UI available */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Database or embedded UI unavailable */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    health: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Database reachable */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Database unavailable */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    getDashboard: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Current committed dashboard */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Board"];
                };
            };
        };
    };
    saveDashboard: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["Board"];
            };
        };
        responses: {
            /** @description Committed dashboard with incremented revision */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Board"];
                };
            };
            400: components["responses"]["ValidationError"];
            /** @description Missing or cross-site Origin */
            403: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            /** @description Revision conflict; preserve local draft and fetch latest */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    validateDashboard: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["Board"];
            };
        };
        responses: {
            /** @description Validated dashboard, not persisted */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Board"];
                };
            };
            400: components["responses"]["ValidationError"];
            /** @description Missing or cross-site Origin */
            403: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
        };
    };
    getSnapshots: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /**
             * @description Immediate map from source ID to cached snapshot. Reading expired
             *     sources schedules bounded asynchronous refresh; it never waits for
             *     provider requests. Loading/error states may have no data.
             */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": {
                        [key: string]: components["schemas"]["Snapshot"];
                    };
                };
            };
        };
    };
}
