import { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { api } from "../lib/api";

export default function Stock() {
  const { token } = useAuth();
  const [stocks, setStocks] = useState([]);
  const [products, setProducts] = useState([]);
  const [locations, setLocations] = useState([]);
  const [form, setForm] = useState({ productId: "", locationId: "", quantity: "" });
  const [locationForm, setLocationForm] = useState({ code: "", description: "" });
  const [error, setError] = useState("");

  async function load() {
    const [s, p, l] = await Promise.all([
      api.get("/stock", token),
      api.get("/products", token),
      api.get("/locations", token),
    ]);
    setStocks(s);
    setProducts(p);
    setLocations(l);
  }

  useEffect(() => {
    load().catch((err) => setError(err.message));
  }, [token]);

  async function handleAdjust(e) {
    e.preventDefault();
    setError("");
    try {
      await api.post(
        "/stock/adjust",
        { ...form, quantity: Number(form.quantity) },
        token
      );
      setForm({ productId: "", locationId: "", quantity: "" });
      await load();
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleCreateLocation(e) {
    e.preventDefault();
    setError("");
    try {
      await api.post("/locations", locationForm, token);
      setLocationForm({ code: "", description: "" });
      await load();
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div>
      <h1>Estoque</h1>

      <section>
        <h2>Nova localização</h2>
        <form className="inline-form" onSubmit={handleCreateLocation}>
          <input
            placeholder="Código (ex: A1-01)"
            value={locationForm.code}
            onChange={(e) => setLocationForm({ ...locationForm, code: e.target.value })}
            required
          />
          <input
            placeholder="Descrição"
            value={locationForm.description}
            onChange={(e) => setLocationForm({ ...locationForm, description: e.target.value })}
          />
          <button type="submit">Adicionar</button>
        </form>
      </section>

      <section>
        <h2>Ajustar estoque</h2>
        <form className="inline-form" onSubmit={handleAdjust}>
          <select
            value={form.productId}
            onChange={(e) => setForm({ ...form, productId: e.target.value })}
            required
          >
            <option value="">Produto</option>
            {products.map((p) => (
              <option key={p.id} value={p.id}>{p.sku} - {p.name}</option>
            ))}
          </select>
          <select
            value={form.locationId}
            onChange={(e) => setForm({ ...form, locationId: e.target.value })}
            required
          >
            <option value="">Localização</option>
            {locations.map((l) => (
              <option key={l.id} value={l.id}>{l.code}</option>
            ))}
          </select>
          <input
            type="number"
            placeholder="Quantidade (+/-)"
            value={form.quantity}
            onChange={(e) => setForm({ ...form, quantity: e.target.value })}
            required
          />
          <button type="submit">Ajustar</button>
        </form>
      </section>

      {error && <p className="error">{error}</p>}

      <table>
        <thead>
          <tr>
            <th>Produto</th>
            <th>Localização</th>
            <th>Quantidade</th>
          </tr>
        </thead>
        <tbody>
          {stocks.map((s) => (
            <tr key={s.id}>
              <td>{s.product.sku} - {s.product.name}</td>
              <td>{s.location.code}</td>
              <td>{s.quantity}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
