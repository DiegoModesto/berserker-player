import SwiftUI

struct LoginView: View {
    @Environment(Session.self) private var session
    @State private var server = UserDefaults.standard.string(forKey: "serverURL") ?? ""
    @State private var username = ""
    @State private var password = ""

    var body: some View {
        VStack(spacing: 20) {
            Spacer()
            Image(systemName: "waveform.circle.fill")
                .font(.system(size: 64)).foregroundStyle(Theme.accent)
            Text("BerserkerPlayer").font(.largeTitle.bold())

            VStack(spacing: 12) {
                TextField("Servidor (ex.: http://192.168.0.10:4533)", text: $server)
                    .textContentType(.URL).keyboardType(.URL)
                    .autocorrectionDisabled().textInputAutocapitalization(.never)
                TextField("Usuário", text: $username)
                    .textContentType(.username)
                    .autocorrectionDisabled().textInputAutocapitalization(.never)
                SecureField("Senha", text: $password)
                    .textContentType(.password)
            }
            .textFieldStyle(.roundedBorder)

            if let err = session.lastError {
                Text(err).font(.footnote).foregroundStyle(.red)
            }

            Button {
                Task { await session.login(server: server, username: username, password: password) }
            } label: {
                Text("Entrar").frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .disabled(server.isEmpty || username.isEmpty || password.isEmpty)

            Spacer()
        }
        .padding()
    }
}
