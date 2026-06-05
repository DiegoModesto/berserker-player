import Foundation

/// Lógica pura da fila de reprodução (sem AVFoundation), testável isoladamente.
struct PlaybackQueue {
    enum RepeatMode { case off, all, one }

    var songs: [Song] = []
    var index: Int = 0
    var repeatMode: RepeatMode = .off
    var shuffle: Bool = false

    var current: Song? { songs.indices.contains(index) ? songs[index] : nil }
    var isEmpty: Bool { songs.isEmpty }

    /// Próximo índice a tocar; `nil` significa "fim da fila" (parar).
    /// `randomIndex` é injetável para tornar o shuffle determinístico em testes.
    func nextIndex(randomIndex: (Int) -> Int = { Int.random(in: 0..<$0) }) -> Int? {
        guard !songs.isEmpty else { return nil }
        if repeatMode == .one { return index }
        if shuffle { return randomIndex(songs.count) }
        if index < songs.count - 1 { return index + 1 }
        return repeatMode == .all ? 0 : nil
    }

    /// Índice anterior. Se `currentTime > restartThreshold`, reinicia a faixa atual.
    func previousIndex(currentTime: Double, restartThreshold: Double = 3) -> Int {
        if currentTime > restartThreshold { return index }
        return max(0, index - 1)
    }
}
