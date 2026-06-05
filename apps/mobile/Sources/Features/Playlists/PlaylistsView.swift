import SwiftUI

struct PlaylistsView: View {
    @Environment(Session.self) private var session
    @State private var playlists: [Playlist] = []
    @State private var loaded = false

    var body: some View {
        NavigationStack {
            List(playlists) { pl in
                NavigationLink(value: pl) {
                    HStack {
                        Image(systemName: "music.note.list").foregroundStyle(Theme.accent)
                        Text(pl.name)
                        Spacer()
                        Text("\(pl.songCount ?? 0)").foregroundStyle(.secondary).font(.caption)
                    }
                }
            }
            .overlay {
                if loaded && playlists.isEmpty {
                    ContentUnavailableView("Sem playlists", systemImage: "music.note.list")
                }
            }
            .navigationTitle("Playlists")
            .navigationDestination(for: Playlist.self) { PlaylistDetailView(playlistID: $0.id, title: $0.name) }
        }
        .task {
            playlists = (try? await session.api.playlists()) ?? []
            loaded = true
        }
    }
}

struct PlaylistDetailView: View {
    let playlistID: String
    let title: String
    @Environment(Session.self) private var session
    @Environment(PlaybackEngine.self) private var playback
    @State private var detail: PlaylistDetail?

    var body: some View {
        List {
            if let detail {
                ForEach(Array(detail.songs.enumerated()), id: \.element.id) { idx, song in
                    Button {
                        playback.play(songs: detail.songs, startAt: idx)
                    } label: {
                        HStack {
                            Text("\(idx + 1)").font(.caption.monospacedDigit())
                                .foregroundStyle(.secondary).frame(width: 24)
                            VStack(alignment: .leading) {
                                Text(song.title).lineLimit(1)
                                Text(song.artistName ?? "").font(.caption).foregroundStyle(.secondary)
                            }
                            Spacer()
                            Text((song.duration ?? 0).asTimeString)
                                .font(.caption.monospacedDigit()).foregroundStyle(.secondary)
                        }
                    }
                    .buttonStyle(.plain)
                }
            } else {
                ProgressView()
            }
        }
        .navigationTitle(title)
        .navigationBarTitleDisplayMode(.inline)
        .task { detail = try? await session.api.playlistDetail(playlistID) }
    }
}
