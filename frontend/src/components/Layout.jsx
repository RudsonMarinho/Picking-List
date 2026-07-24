import { NavLink, Outlet } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

export default function Layout() {
  const { user, logout } = useAuth();

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">Picking List</div>
        <nav>
          <NavLink to="/" end>Dashboard</NavLink>
          <NavLink to="/products">Produtos</NavLink>
          <NavLink to="/stock">Estoque</NavLink>
          <NavLink to="/orders">Pedidos</NavLink>
          <NavLink to="/picking-lists">Picking Lists</NavLink>
        </nav>
        <div className="user-info">
          <span>{user?.name}</span>
          <button onClick={logout}>Sair</button>
        </div>
      </header>
      <main className="content">
        <Outlet />
      </main>
    </div>
  );
}
