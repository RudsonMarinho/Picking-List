import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { api } from "../lib/api";

export default function Orders() {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [orders, setOrders] = useState([]);
  const [products, setProducts] = useState([]);
  const [orderNumber, setOrderNumber] = useState("");
  const [items, setItems] = useState([{ productId: "", quantity: "" }]);
  const [error, setError] = useState("");

  async function load() {
    const [o, p] = await Promise.all([api.get("/orders", token), api.get("/products", token)]);
    setOrders(o);
    setProducts(p);
  }

  useEffect(() => {
    load().catch((err) => setError(err.message));
  }, [token]);

  function updateItem(index, field, value) {
    const next = [...items];
    next[index] = { ...next[index], [field]: value };
    setItems(next);
  }

  function addItemRow() {
    setItems([...items, { productId: "", quantity: "" }]);
  }

  function removeItemRow(index) {
    setItems(items.filter((_, i) => i !== index));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setError("");
    try {
      await api.post(
        "/orders",
        {
          orderNumber,
          items: items.map((i) => ({ productId: i.productId, quantity: Number(i.quantity) })),
        },
        token
      );
      setOrderNumber("");
      setItems([{ productId: "", quantity: "" }]);
      await load();
    } catch (err) {
      setError(err.message);
    }
  }

  async function handleGeneratePickingList(orderId) {
    setError("");
    try {
      const pickingList = await api.post(`/picking-lists/generate/${orderId}`, {}, token);
      navigate(`/picking-lists/${pickingList.id}`);
    } catch (err) {
      setError(err.message);
    }
  }

  return (
    <div>
      <h1>Pedidos</h1>

      <section>
        <h2>Novo pedido</h2>
        <form onSubmit={handleSubmit}>
          <input
            placeholder="Número do pedido"
            value={orderNumber}
            onChange={(e) => setOrderNumber(e.target.value)}
            required
          />

          {items.map((item, index) => (
            <div className="inline-form" key={index}>
              <select
                value={item.productId}
                onChange={(e) => updateItem(index, "productId", e.target.value)}
                required
              >
                <option value="">Produto</option>
                {products.map((p) => (
                  <option key={p.id} value={p.id}>{p.sku} - {p.name}</option>
                ))}
              </select>
              <input
                type="number"
                placeholder="Quantidade"
                value={item.quantity}
                onChange={(e) => updateItem(index, "quantity", e.target.value)}
                required
              />
              {items.length > 1 && (
                <button type="button" onClick={() => removeItemRow(index)}>Remover</button>
              )}
            </div>
          ))}

          <button type="button" onClick={addItemRow}>+ Item</button>
          <button type="submit">Criar pedido</button>
        </form>
      </section>

      {error && <p className="error">{error}</p>}

      <table>
        <thead>
          <tr>
            <th>Número</th>
            <th>Status</th>
            <th>Itens</th>
            <th>Ações</th>
          </tr>
        </thead>
        <tbody>
          {orders.map((o) => (
            <tr key={o.id}>
              <td>{o.orderNumber}</td>
              <td>{o.status}</td>
              <td>{o.items.length}</td>
              <td>
                {o.pickingList ? (
                  <button onClick={() => navigate(`/picking-lists/${o.pickingList.id}`)}>
                    Ver picking list
                  </button>
                ) : (
                  <button onClick={() => handleGeneratePickingList(o.id)}>Gerar picking list</button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
