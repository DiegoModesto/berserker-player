import SwiftUI

struct MiniPlayerBar: View {
    @Environment(PlaybackEngine.self) private var playback
    @State private var showFull = false

    var body: some View {
        if let song = playback.current {
            HStack(spacing: 12) {
                CoverImage(coverArtId: song.coverArtId, size: 40)
                VStack(alignment: .leading, spacing: 2) {
                    Text(song.title).font(.subheadline).lineLimit(1)
                    Text(song.artistName ?? "").font(.caption).foregroundStyle(.secondary).lineLimit(1)
                }
                Spacer()
                Button { playback.togglePlayPause() } label: {
                    Image(systemName: playback.isPlaying ? "pause.fill" : "play.fill").font(.title3)
                }
                Button { playback.next() } label: {
                    Image(systemName: "forward.fill").font(.title3)
                }
            }
            .padding(.horizontal, 12).padding(.vertical, 8)
            .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: 12))
            .padding(.horizontal, 8)
            .onTapGesture { showFull = true }
            .sheet(isPresented: $showFull) { NowPlayingView() }
        }
    }
}

struct NowPlayingView: View {
    @Environment(PlaybackEngine.self) private var playback
    @Environment(Session.self) private var session
    @Environment(\.dismiss) private var dismiss
    @State private var scrubbing = false
    @State private var scrubValue: Double = 0
    @State private var starred = false

    var body: some View {
        VStack(spacing: 24) {
            Capsule().fill(.secondary).frame(width: 40, height: 5).padding(.top, 8)
            Spacer()
            CoverImage(coverArtId: playback.current?.coverArtId, size: 300)
                .shadow(radius: 12)
            VStack(spacing: 4) {
                Text(playback.current?.title ?? "").font(.title2.bold()).lineLimit(1)
                Text(playback.current?.artistName ?? "").foregroundStyle(.secondary)
            }

            VStack(spacing: 4) {
                Slider(value: Binding(
                    get: { scrubbing ? scrubValue : playback.currentTime },
                    set: { scrubValue = $0 }
                ), in: 0...max(playback.duration, 1), onEditingChanged: { editing in
                    scrubbing = editing
                    if !editing { playback.seek(to: scrubValue) }
                })
                HStack {
                    Text(playback.currentTime.asTimeString).font(.caption.monospacedDigit())
                    Spacer()
                    Text(playback.duration.asTimeString).font(.caption.monospacedDigit())
                }.foregroundStyle(.secondary)
            }.padding(.horizontal)

            HStack(spacing: 40) {
                Button { playback.previous() } label: { Image(systemName: "backward.fill").font(.title) }
                Button { playback.togglePlayPause() } label: {
                    Image(systemName: playback.isPlaying ? "pause.circle.fill" : "play.circle.fill")
                        .font(.system(size: 64))
                }
                Button { playback.next() } label: { Image(systemName: "forward.fill").font(.title) }
            }
            .tint(Theme.accent)

            HStack(spacing: 50) {
                Button { playback.shuffle.toggle() } label: {
                    Image(systemName: "shuffle").foregroundStyle(playback.shuffle ? Theme.accent : .secondary)
                }
                Button { toggleStar() } label: {
                    Image(systemName: starred ? "heart.fill" : "heart")
                        .foregroundStyle(starred ? Theme.accent : .secondary)
                }
                Button { cycleRepeat() } label: {
                    Image(systemName: playback.repeatMode == .one ? "repeat.1" : "repeat")
                        .foregroundStyle(playback.repeatMode == .off ? .secondary : Theme.accent)
                }
            }
            Spacer()
        }
        .padding()
        .task(id: playback.current?.id) {
            starred = playback.current?.starred ?? false
        }
    }

    private func toggleStar() {
        guard let song = playback.current else { return }
        let next = !starred
        starred = next
        Task { try? await session.api.star(id: song.id, type: "song", on: next) }
    }

    private func cycleRepeat() {
        switch playback.repeatMode {
        case .off: playback.repeatMode = .all
        case .all: playback.repeatMode = .one
        case .one: playback.repeatMode = .off
        }
    }
}
