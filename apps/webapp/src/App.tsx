import { Navigate, Route, Routes } from "react-router-dom";
import { useSession } from "./auth/session";
import { Layout } from "./components/Layout";
import { AlbumPage } from "./features/album/AlbumPage";
import { LibraryPage } from "./features/library/LibraryPage";
import { LoginPage } from "./features/login/LoginPage";
import { SearchPage } from "./features/search/SearchPage";

export function App() {
  const { status } = useSession();

  if (status === "loading") {
    return (
      <div className="h-screen flex items-center justify-center bg-neutral-950 text-neutral-400">
        Carregando…
      </div>
    );
  }

  if (status === "anonymous") return <LoginPage />;

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<LibraryPage />} />
        <Route path="/album/:id" element={<AlbumPage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
