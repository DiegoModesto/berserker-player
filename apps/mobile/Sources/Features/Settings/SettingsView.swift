import SwiftUI

struct SettingsView: View {
    @Environment(Session.self) private var session

    var body: some View {
        NavigationStack {
            Form {
                Section("Conta") {
                    LabeledContent("Usuário", value: session.user?.username ?? "—")
                    LabeledContent("Servidor", value: session.serverURLString)
                    if session.user?.isAdmin == true {
                        Label("Administrador", systemImage: "checkmark.seal.fill")
                            .foregroundStyle(Theme.accent)
                    }
                }
                Section {
                    NavigationLink {
                        DownloadsView()
                    } label: {
                        Label("Downloads", systemImage: "arrow.down.circle")
                    }
                }
                Section {
                    Button(role: .destructive) {
                        Task { await session.logout() }
                    } label: {
                        Text("Sair")
                    }
                }
                Section {
                    LabeledContent("Versão", value: "0.1.0")
                }
            }
            .navigationTitle("Ajustes")
        }
    }
}
