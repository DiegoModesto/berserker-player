import SwiftUI

enum Theme {
    static let accent = Color(red: 0.78, green: 0.10, blue: 0.10) // vermelho "berserker"
    static let cardCorner: CGFloat = 8
}

/// Carrega uma capa via APIClient (URL com token de mídia), com placeholder.
struct CoverImage: View {
    let coverArtId: String?
    var size: CGFloat = 160

    @Environment(Session.self) private var session
    @State private var url: URL?

    var body: some View {
        Group {
            if let url {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let img): img.resizable().scaledToFill()
                    default: placeholder
                    }
                }
            } else {
                placeholder
            }
        }
        .frame(width: size, height: size)
        .clipped()
        .clipShape(RoundedRectangle(cornerRadius: Theme.cardCorner))
        .task(id: coverArtId) {
            guard let id = coverArtId else { return }
            url = await session.api.coverURL(coverArtId: id, size: Int(size * 2))
        }
    }

    private var placeholder: some View {
        ZStack {
            RoundedRectangle(cornerRadius: Theme.cardCorner).fill(.gray.opacity(0.25))
            Image(systemName: "music.note").font(.title2).foregroundStyle(.secondary)
        }
    }
}

extension Int {
    /// Segundos → "m:ss".
    var asTimeString: String {
        let m = self / 60, s = self % 60
        return String(format: "%d:%02d", m, s)
    }
}

extension Double {
    var asTimeString: String { Int(self.rounded()).asTimeString }
}
