import SwiftUI

@main
struct BerserkerApp: App {
    @State private var session = Session()
    @State private var playback: PlaybackEngine

    init() {
        let s = Session()
        _session = State(initialValue: s)
        _playback = State(initialValue: PlaybackEngine(api: s.api))
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environment(session)
                .environment(playback)
                .tint(Theme.accent)
                .task { await session.restoreIfPossible() }
        }
    }
}
