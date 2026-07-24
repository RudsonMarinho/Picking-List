import { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { api } from "../lib/api";

export default function Dashboard() {
  const { token } = useAuth();
  const [summary, setSummary] = useState({ products: 0, orders: 0, pickingLists: 0 });
  const [error, setError] = useState("");

  useEffect(() => {
    async function load() {
      try {
        const [products, orders, pickingLists] = await Promise.all([
          api.get("/products", token),
          api.get("/orders", token),
          api.get("/picking-lists", token),
        ]);
        setSummary({
          products: products.length,
          orders: orders.length,
          pickingLists: pickingLists.filter((p) => p.status !== "COMPLETED").length,
        });
      } catch (err) {
        setError(err.message);
      }
    }
    load();
  }, [token]);

  return (
    <div>
      <h1>Dashboard</h1>
      {error && <p className="error">{error}</p>}
      <div className="cards">
        <div className="card">
          <span className="card-value">{summary.products}</span>
          <span className="card-label">Produtos cadastrados</span>
        </div>
        <div className="card">
          <span className="card-value">{summary.orders}</span>
          <span className="card-label">Pedidos</span>
        </div>
        <div className="card">
          <span className="card-value">{summary.pickingLists}</span>
          <span className="card-label">Picking lists em aberto</span>
        </div>
      </div>
    </div>
  );
}
