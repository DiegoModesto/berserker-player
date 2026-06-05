import SwiftUI

struct AlbumDetailView: View {
    let albumID: String
    @Environment(Session.self) private var session
    @Environment(PlaybackEngine.self) private var playback
    @Environment(DownloadManager.self) private var downloads
    @State private var detail: AlbumDetail?

    var body: some View {
        List {
            if let detail {
                Section {
                    HStack(alignment: .top, spacing: 16) {
                        CoverImage(coverArtId: detail.coverArtId ?? detail.id, size: 120)
                        VStack(alignment: .leading, spacing: 4) {
                            Text(detail.name).font(.title3.bold())
                            Text(detail.artistName ?? "").foregroundStyle(.secondary)
                            if let y = detail.year, y > 0 { Text(String(y)).font(.caption).foregroundStyle(.secondary) }
                            Button {
                                playback.play(songs: detail.songs, startAt: 0)
                            } label: {
                                Label("Tocar", systemImage: "play.fill")
                            }
                            .buttonStyle(.borderedProminent).controlSize(.small)
                            .padding(.top, 4)
                        }
                        Spacer()
                    }
                }
                Section {
                    ForEach(Array(detail.songs.enumerated()), id: \.element.id) { idx, song in
                        HStack {
                            Button {
                                playback.play(songs: detail.songs, startAt: idx)
                            } label: {
                                HStack {
                                    Text("\(song.track ?? idx + 1)")
                                        .font(.caption.monospacedDigit()).foregroundStyle(.secondary).frame(width: 24)
                                    Text(song.title).lineLimit(1)
                                    Spacer()
                                }
                            }
                            .buttonStyle(.plain)
                            DownloadButton(song: song)
                            Text((song.duration ?? 0).asTimeString)
                                .font(.caption.monospacedDigit()).foregroundStyle(.secondary)
                        }
                    }
                }
            } else {
                ProgressView()
            }
        }
        .navigationTitle(detail?.name ?? "Álbum")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            detail = try? await session.api.albumDetail(albumID)
        }
    }
}

/// Botão de download/estado offline de uma faixa.
struct DownloadButton: View {
    let song: Song
    @Environment(DownloadManager.self) private var downloads
    @State private var downloaded = false

    var body: some View {
        Group {
            if downloads.inProgress.contains(song.id) {
                ProgressView()
            } else if downloaded {
                Image(systemName: "checkmark.circle.fill").foregroundStyle(Theme.accent)
            } else {
                Button {
                    Task {
                        await downloads.download(song)
                        downloaded = downloads.isDownloaded(song.id)
                    }
                } label: {
                    Image(systemName: "arrow.down.circle")
                }
                .buttonStyle(.borderless)
            }
        }
        .task(id: song.id) { downloaded = downloads.isDownloaded(song.id) }
    }
}
