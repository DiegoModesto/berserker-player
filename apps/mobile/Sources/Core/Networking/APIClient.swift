import Foundation

/// Cliente da API nativa: autenticação por Bearer, refresh automático em 401,
/// e token de mídia assinado (cacheado) para URLs de /stream e /cover.
actor APIClient {
    enum APIError: Error { case notAuthenticated, server(Int), decoding, badURL }

    private var baseURL: URL
    private var accessToken: String?
    private var refreshToken: String?

    private var mediaToken: String?
    private var mediaTokenExpiry: Date = .distantPast

    private let session: URLSession
    private let decoder = JSONDecoder()

    init(baseURL: URL) {
        self.baseURL = baseURL
        let cfg = URLSessionConfiguration.default
        cfg.waitsForConnectivity = true
        self.session = URLSession(configuration: cfg)
        self.refreshToken = KeychainStore.get("refreshToken")
    }

    func updateBaseURL(_ url: URL) { self.baseURL = url }
    var currentBaseURL: URL { baseURL }
    var hasSession: Bool { refreshToken != nil }

    // MARK: - Auth

    func login(username: String, password: String) async throws {
        let body = try JSONSerialization.data(withJSONObject: ["username": username, "password": password])
        var req = request("/api/v1/auth/login", method: "POST")
        req.httpBody = body
        let (data, resp) = try await session.data(for: req)
        guard (resp as? HTTPURLResponse)?.statusCode == 200 else { throw APIError.server(statusCode(resp)) }
        let tp = try decoder.decode(TokenPair.self, from: data)
        applyTokens(tp)
    }

    func logout() {
        accessToken = nil
        refreshToken = nil
        mediaToken = nil
        mediaTokenExpiry = .distantPast
        KeychainStore.delete("refreshToken")
    }

    private func applyTokens(_ tp: TokenPair) {
        accessToken = tp.accessToken
        if let rt = tp.refreshToken {
            refreshToken = rt
            KeychainStore.set(rt, for: "refreshToken")
        }
    }

    @discardableResult
    private func refresh() async throws -> Bool {
        guard let rt = refreshToken else { throw APIError.notAuthenticated }
        var req = request("/api/v1/auth/refresh", method: "POST")
        req.httpBody = try JSONSerialization.data(withJSONObject: ["refreshToken": rt])
        let (data, resp) = try await session.data(for: req)
        guard (resp as? HTTPURLResponse)?.statusCode == 200 else {
            logout()
            throw APIError.notAuthenticated
        }
        let tp = try decoder.decode(TokenPair.self, from: data)
        applyTokens(tp)
        return true
    }

    // MARK: - Requests

    private func request(_ path: String, method: String = "GET") -> URLRequest {
        var req = URLRequest(url: baseURL.appendingPathComponent(path))
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        return req
    }

    /// Executa uma chamada autenticada, com retry transparente após refresh em 401.
    private func authedData(for path: String, method: String = "GET", body: Data? = nil) async throws -> Data {
        if accessToken == nil { _ = try await refresh() }
        func attempt() async throws -> (Data, URLResponse) {
            var req = request(path, method: method)
            req.httpBody = body
            if let t = accessToken { req.setValue("Bearer \(t)", forHTTPHeaderField: "Authorization") }
            return try await session.data(for: req)
        }
        var (data, resp) = try await attempt()
        if statusCode(resp) == 401 {
            _ = try await refresh()
            (data, resp) = try await attempt()
        }
        let code = statusCode(resp)
        guard (200..<300).contains(code) else { throw APIError.server(code) }
        return data
    }

    private func get<T: Decodable>(_ path: String, as type: T.Type) async throws -> T {
        let data = try await authedData(for: path)
        do { return try decoder.decode(T.self, from: data) }
        catch { throw APIError.decoding }
    }

    private func statusCode(_ resp: URLResponse) -> Int { (resp as? HTTPURLResponse)?.statusCode ?? 0 }

    // MARK: - Library

    func me() async throws -> User { try await get("/api/v1/me", as: User.self) }

    func albums(filter: String = "all", offset: Int = 0, limit: Int = 50,
                sort: String = "name") async throws -> Page<Album> {
        try await get("/api/v1/albums?filter=\(filter)&offset=\(offset)&limit=\(limit)&sort=\(sort)", as: Page<Album>.self)
    }

    func albumDetail(_ id: String) async throws -> AlbumDetail {
        try await get("/api/v1/albums/\(id)", as: AlbumDetail.self)
    }

    func artists(offset: Int = 0, limit: Int = 100) async throws -> Page<Artist> {
        try await get("/api/v1/artists?offset=\(offset)&limit=\(limit)", as: Page<Artist>.self)
    }

    func artistDetail(_ id: String) async throws -> ArtistDetail {
        try await get("/api/v1/artists/\(id)", as: ArtistDetail.self)
    }

    func search(_ query: String) async throws -> SearchResult {
        let q = query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? ""
        return try await get("/api/v1/search?q=\(q)", as: SearchResult.self)
    }

    func playlists() async throws -> [Playlist] { try await get("/api/v1/playlists", as: [Playlist].self) }

    func playlistDetail(_ id: String) async throws -> PlaylistDetail {
        try await get("/api/v1/playlists/\(id)", as: PlaylistDetail.self)
    }

    // MARK: - Annotations

    func star(id: String, type: String, on: Bool) async throws {
        let body = try JSONEncoder().encode(ItemRef(id: id, type: type))
        _ = try await authedData(for: "/api/v1/star", method: on ? "POST" : "DELETE", body: body)
    }

    func scrobble(songId: String, event: String) async throws {
        let body = try JSONSerialization.data(withJSONObject: ["songId": songId, "event": event])
        _ = try await authedData(for: "/api/v1/scrobble", method: "POST", body: body)
    }

    // MARK: - Media tokens & URLs

    private func ensureMediaToken() async throws -> String {
        if let t = mediaToken, mediaTokenExpiry.timeIntervalSinceNow > 60 { return t }
        let data = try await authedData(for: "/api/v1/auth/media-token", method: "POST")
        let mt = try decoder.decode(MediaTokenResponse.self, from: data)
        mediaToken = mt.token
        let fmt = ISO8601DateFormatter()
        fmt.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        mediaTokenExpiry = fmt.date(from: mt.expiresAt) ?? Date().addingTimeInterval(300)
        return mt.token
    }

    func streamURL(songId: String) async throws -> URL {
        let token = try await ensureMediaToken()
        var comps = URLComponents(url: baseURL.appendingPathComponent("/api/v1/stream/\(songId)"),
                                  resolvingAgainstBaseURL: false)!
        comps.queryItems = [URLQueryItem(name: "token", value: token)]
        guard let url = comps.url else { throw APIError.badURL }
        return url
    }

    func coverURL(coverArtId: String, size: Int = 300) async -> URL? {
        guard let token = try? await ensureMediaToken() else { return nil }
        var comps = URLComponents(url: baseURL.appendingPathComponent("/api/v1/cover/\(coverArtId)"),
                                  resolvingAgainstBaseURL: false)!
        comps.queryItems = [
            URLQueryItem(name: "token", value: token),
            URLQueryItem(name: "size", value: String(size)),
        ]
        return comps.url
    }
}
