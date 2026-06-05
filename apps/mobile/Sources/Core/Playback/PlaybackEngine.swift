import Foundation
import AVFoundation
import MediaPlayer
import Observation
import UIKit

/// Engine de reprodução: fila (AVQueuePlayer), áudio em background,
/// Now Playing (tela de bloqueio/CarPlay) e comandos remotos.
@Observable
@MainActor
final class PlaybackEngine {
    typealias RepeatMode = PlaybackQueue.RepeatMode

    private(set) var queue: [Song] = []
    private(set) var currentIndex: Int = 0
    private(set) var isPlaying: Bool = false
    private(set) var currentTime: Double = 0
    private(set) var duration: Double = 0
    var repeatMode: RepeatMode = .off
    var shuffle: Bool = false

    var current: Song? { queue.indices.contains(currentIndex) ? queue[currentIndex] : nil }

    private let api: APIClient
    /// Resolve uma URL local para reprodução offline (injetado opcionalmente).
    var localURLProvider: ((String) -> URL?)?
    private var player = AVQueuePlayer()
    private var timeObserver: Any?
    private var scrobbledCurrent = false

    private var snapshot: PlaybackQueue {
        PlaybackQueue(songs: queue, index: currentIndex, repeatMode: repeatMode, shuffle: shuffle)
    }

    init(api: APIClient) {
        self.api = api
        configureAudioSession()
        setupRemoteCommands()
        observeTime()
        NotificationCenter.default.addObserver(
            self, selector: #selector(handleItemEnded),
            name: .AVPlayerItemDidPlayToEndTime, object: nil)
    }

    // MARK: - Public controls

    func play(songs: [Song], startAt index: Int = 0) {
        queue = songs
        currentIndex = max(0, min(index, songs.count - 1))
        Task { await loadCurrent(autoplay: true) }
    }

    func togglePlayPause() {
        if isPlaying { player.pause() } else { player.play() }
        isPlaying.toggle()
        updateNowPlaying()
    }

    func next() {
        guard !queue.isEmpty else { return }
        if repeatMode == .one {
            seek(to: 0)
            player.play()
            return
        }
        guard let nextIdx = snapshot.nextIndex() else { return }
        currentIndex = nextIdx
        Task { await loadCurrent(autoplay: true) }
    }

    func previous() {
        guard !queue.isEmpty else { return }
        if currentTime > 3 {
            seek(to: 0)
            return
        }
        currentIndex = snapshot.previousIndex(currentTime: currentTime)
        Task { await loadCurrent(autoplay: true) }
    }

    func seek(to seconds: Double) {
        player.seek(to: CMTime(seconds: seconds, preferredTimescale: 600))
        currentTime = seconds
        updateNowPlaying()
    }

    // MARK: - Loading

    private func loadCurrent(autoplay: Bool) async {
        guard let song = current else { return }
        do {
            // Reprodução offline: usa arquivo local quando disponível.
            let url: URL
            if let local = localURLProvider?(song.id) {
                url = local
            } else {
                url = try await api.streamURL(songId: song.id)
            }
            let item = AVPlayerItem(url: url)
            player.removeAllItems()
            player.insert(item, after: nil)
            scrobbledCurrent = false
            duration = Double(song.duration ?? 0)
            if autoplay { player.play(); isPlaying = true }
            await updateArtwork(for: song)
            updateNowPlaying()
            try? await api.scrobble(songId: song.id, event: "nowplaying")
        } catch {
            isPlaying = false
        }
    }

    @objc private func handleItemEnded() {
        Task { @MainActor in self.next() }
    }

    // MARK: - Time observation & scrobble

    private func observeTime() {
        let interval = CMTime(seconds: 0.5, preferredTimescale: 600)
        timeObserver = player.addPeriodicTimeObserver(forInterval: interval, queue: .main) { [weak self] time in
            guard let self else { return }
            Task { @MainActor in
                self.currentTime = time.seconds
                if let item = self.player.currentItem, item.duration.seconds.isFinite, item.duration.seconds > 0 {
                    self.duration = item.duration.seconds
                }
                self.maybeScrobble()
            }
        }
    }

    private func maybeScrobble() {
        guard !scrobbledCurrent, duration > 0, let song = current else { return }
        if currentTime / duration >= 0.5 {
            scrobbledCurrent = true
            Task { try? await api.scrobble(songId: song.id, event: "submission") }
        }
    }

    // MARK: - Audio session / Now Playing / Remote

    private func configureAudioSession() {
        let session = AVAudioSession.sharedInstance()
        try? session.setCategory(.playback, mode: .default)
        try? session.setActive(true)
    }

    private func setupRemoteCommands() {
        let c = MPRemoteCommandCenter.shared()
        c.playCommand.addTarget { [weak self] _ in self?.resume(); return .success }
        c.pauseCommand.addTarget { [weak self] _ in self?.pause(); return .success }
        c.nextTrackCommand.addTarget { [weak self] _ in self?.next(); return .success }
        c.previousTrackCommand.addTarget { [weak self] _ in self?.previous(); return .success }
        c.changePlaybackPositionCommand.addTarget { [weak self] event in
            guard let e = event as? MPChangePlaybackPositionCommandEvent else { return .commandFailed }
            self?.seek(to: e.positionTime); return .success
        }
    }

    private func resume() { player.play(); isPlaying = true; updateNowPlaying() }
    private func pause() { player.pause(); isPlaying = false; updateNowPlaying() }

    private func updateNowPlaying() {
        guard let song = current else { return }
        var info = MPNowPlayingInfoCenter.default().nowPlayingInfo ?? [:]
        info[MPMediaItemPropertyTitle] = song.title
        info[MPMediaItemPropertyArtist] = song.artistName ?? ""
        info[MPMediaItemPropertyAlbumTitle] = song.albumName ?? ""
        info[MPMediaItemPropertyPlaybackDuration] = duration
        info[MPNowPlayingInfoPropertyElapsedPlaybackTime] = currentTime
        info[MPNowPlayingInfoPropertyPlaybackRate] = isPlaying ? 1.0 : 0.0
        MPNowPlayingInfoCenter.default().nowPlayingInfo = info
    }

    private func updateArtwork(for song: Song) async {
        guard let coverId = song.coverArtId, let url = await api.coverURL(coverArtId: coverId, size: 600) else { return }
        guard let (data, _) = try? await URLSession.shared.data(from: url),
              let image = UIImage(data: data) else { return }
        let artwork = MPMediaItemArtwork(boundsSize: image.size) { _ in image }
        var info = MPNowPlayingInfoCenter.default().nowPlayingInfo ?? [:]
        info[MPMediaItemPropertyArtwork] = artwork
        MPNowPlayingInfoCenter.default().nowPlayingInfo = info
    }
}
