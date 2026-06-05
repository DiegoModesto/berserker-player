import SwiftUI

struct SearchView: View {
    @Environment(Session.self) private var session
    @Environment(PlaybackEngine.self) private var playback
    @State private var query = ""
    @State private var result: SearchResult?
    @State private var searchTask: Task<Void, Never>?

    var body: some View {
        NavigationStack {
            List {
                if let result {
                    if !result.albums.isEmpty {
                        Section("Álbuns") {
                            ForEach(result.albums) { album in
                                NavigationLink(value: album) {
                                    HStack {
                                        CoverImage(coverArtId: album.coverArtId ?? album.id, size: 44)
                                        VStack(alignment: .leading) {
                                            Text(album.name).lineLimit(1)
                                            Text(album.artistName ?? "").font(.caption).foregroundStyle(.secondary)
                                        }
                                    }
                                }
                            }
                        }
                    }
                    if !result.songs.isEmpty {
                        Section("Faixas") {
                            ForEach(result.songs) { song in
                                Button {
                                    playback.play(songs: [song])
                                } label: {
                                    VStack(alignment: .leading) {
                                        Text(song.title).lineLimit(1)
                                        Text(song.artistName ?? "").font(.caption).foregroundStyle(.secondary)
                                    }
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Buscar")
            .navigationDestination(for: Album.self) { AlbumDetailView(albumID: $0.id) }
        }
        .searchable(text: $query, prompt: "Artistas, álbuns, faixas")
        .onChange(of: query) { _, newValue in
            searchTask?.cancel()
            guard newValue.count >= 2 else { result = nil; return }
            searchTask = Task {
                try? await Task.sleep(for: .milliseconds(300))
                if Task.isCancelled { return }
                result = try? await session.api.search(newValue)
            }
        }
    }
}
