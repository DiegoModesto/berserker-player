import SwiftUI

struct RootView: View {
    @Environment(Session.self) private var session

    var body: some View {
        switch session.state {
        case .loggedIn:
            MainTabView()
        case .authenticating:
            ProgressView("Entrando…")
        case .loggedOut:
            LoginView()
        }
    }
}

struct MainTabView: View {
    var body: some View {
        ZStack(alignment: .bottom) {
            TabView {
                LibraryView()
                    .tabItem { Label("Biblioteca", systemImage: "square.stack") }
                PlaylistsView()
                    .tabItem { Label("Playlists", systemImage: "music.note.list") }
                SearchView()
                    .tabItem { Label("Buscar", systemImage: "magnifyingglass") }
                SettingsView()
                    .tabItem { Label("Ajustes", systemImage: "gearshape") }
            }
            MiniPlayerBar()
                .padding(.bottom, 49) // acima da tab bar
        }
    }
}
