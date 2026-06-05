import SwiftUI
import SwiftData

@main
struct BerserkerApp: App {
    @State private var session: Session
    @State private var playback: PlaybackEngine
    @State private var downloads: DownloadManager?
    private let container: ModelContainer?

    init() {
        let s = Session()
        let engine = PlaybackEngine(api: s.api)
        _session = State(initialValue: s)
        _playback = State(initialValue: engine)

        // Sob testes (host XCTest), NÃO cria ModelContainer: os testes criam o
        // seu próprio container in-memory, e dois containers para a mesma entidade
        // no mesmo processo provocam trap no SwiftData.
        if Self.isTesting {
            container = nil
            _downloads = State(initialValue: nil)
        } else {
            let c = Self.makeContainer()
            let dir = URL.applicationSupportDirectory.appending(path: "Downloads")
            let store = DownloadStore(context: c.mainContext, directory: dir)
            let manager = DownloadManager(api: s.api, store: store)
            engine.localURLProvider = { manager.localURL($0) }
            container = c
            _downloads = State(initialValue: manager)
        }
    }

    /// Cria o container persistente; cai para in-memory se o store em disco falhar.
    private static func makeContainer() -> ModelContainer {
        if let c = try? ModelContainer(for: DownloadedTrack.self) { return c }
        let cfg = ModelConfiguration(isStoredInMemoryOnly: true)
        return try! ModelContainer(for: DownloadedTrack.self, configurations: cfg)
    }

    /// True quando rodando sob XCTest (host de teste).
    static var isTesting: Bool { NSClassFromString("XCTestCase") != nil }

    var body: some Scene {
        WindowGroup {
            if let downloads, let container {
                RootView()
                    .environment(session)
                    .environment(playback)
                    .environment(downloads)
                    .modelContainer(container)
                    .tint(Theme.accent)
                    .task { await session.restoreIfPossible() }
            } else {
                // Sob testes: sem UI (evita SwiftData/AttributeGraph durante os testes).
                Color.clear
            }
        }
    }
}
