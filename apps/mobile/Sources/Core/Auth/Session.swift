import Foundation
import Observation

/// Estado global de sessão/autenticação, observável pela UI.
@Observable
@MainActor
final class Session {
    enum State { case loggedOut, authenticating, loggedIn }

    private(set) var state: State = .loggedOut
    private(set) var user: User?
    var serverURLString: String = UserDefaults.standard.string(forKey: "serverURL") ?? ""
    var lastError: String?

    private(set) var api: APIClient

    init() {
        let url = URL(string: UserDefaults.standard.string(forKey: "serverURL") ?? "http://localhost:4533")
            ?? URL(string: "http://localhost:4533")!
        self.api = APIClient(baseURL: url)
    }

    func restoreIfPossible() async {
        guard await api.hasSession else { return }
        await loadMe()
    }

    func login(server: String, username: String, password: String) async {
        lastError = nil
        state = .authenticating
        guard let url = normalizedURL(server) else {
            lastError = "URL inválida"; state = .loggedOut; return
        }
        UserDefaults.standard.set(url.absoluteString, forKey: "serverURL")
        serverURLString = url.absoluteString
        await api.updateBaseURL(url)
        do {
            try await api.login(username: username, password: password)
            await loadMe()
        } catch {
            lastError = "Falha no login. Verifique servidor e credenciais."
            state = .loggedOut
        }
    }

    private func loadMe() async {
        do {
            user = try await api.me()
            state = .loggedIn
        } catch {
            state = .loggedOut
        }
    }

    func logout() async {
        await api.logout()
        user = nil
        state = .loggedOut
    }

    private func normalizedURL(_ s: String) -> URL? {
        var str = s.trimmingCharacters(in: .whitespaces)
        if str.isEmpty { return nil }
        if !str.hasPrefix("http://") && !str.hasPrefix("https://") { str = "http://" + str }
        if str.hasSuffix("/") { str.removeLast() }
        return URL(string: str)
    }
}
