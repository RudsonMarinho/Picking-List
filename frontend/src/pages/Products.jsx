import { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { api } from "../lib/api";

export default function Products() {
  const { token } = useAuth();
  const [products, setProducts] = useState([]);
  const [form, setForm] = useState({ sku: "", name: "", unit: "UN" });
  const [error, setError] = useState("");

  async function load() {
    setProducts(await api.get("/products", token));
  }

  useEffect(() => {
    load().catch((err) => setError(err.message));
  }, [token]);

  async function handleSubmit(e) {
    e.preventDefault();
    setError("");
    try {
      await api.post("/products", form, token);
      setForm({ sku: "", name: "", unit: "UN" });
      await load();
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div>
      <h1>Produtos</h1>

      <form className="inline-form" onSubmit={handleSubmit}>
        <input
          placeholder="SKU"
          value={form.sku}
          onChange={(e) => setForm({ ...form, sku: e.target.value })}
          required
        />
        <input
          placeholder="Nome"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          required
        />
        <input
          placeholder="Unidade"
          value={form.unit}
          onChange={(e) => setForm({ ...form, unit: e.target.value })}
        />
        <button type="submit">Adicionar</button>
      </form>

      {error && <p className="error">{error}</p>}

      <table>
        <thead>
          <tr>
            <th>SKU</th>
            <th>Nome</th>
            <th>Unidade</th>
          </tr>
        </thead>
        <tbody>
          {products.map((p) => (
            <tr key={p.id}>
              <td>{p.sku}</td>
              <td>{p.name}</td>
              <td>{p.unit}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
