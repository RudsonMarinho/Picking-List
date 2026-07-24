import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { api } from "../lib/api";

export default function PickingListDetail() {
  const { id } = useParams();
  const { token } = useAuth();
  const [pickingList, setPickingList] = useState(null);
  const [error, setError] = useState("");

  async function load() {
    setPickingList(await api.get(`/picking-lists/${id}`, token));
  }

  useEffect(() => {
    load().catch((err) => setError(err.message));
  }, [id, token]);

  async function handlePick(itemId, remaining) {
    setError("");
    try {
      await api.post(`/picking-lists/${id}/items/${itemId}/pick`, { quantityPicked: remaining }, token);
      await load();
    } catch (err) {
      setError(err.message);
    }
  }

  if (!pickingList) return <p>Carregando...</p>;

  return (
    <div>
      <h1>Picking List - Pedido {pickingList.order.orderNumber}</h1>
      <p>Status: <strong>{pickingList.status}</strong></p>

      {error && <p className="error">{error}</p>}

      <table>
        <thead>
          <tr>
            <th>Produto</th>
            <th>Localização</th>
            <th>Solicitado</th>
            <th>Separado</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {pickingList.items.map((item) => {
            const remaining = item.quantityRequested - item.quantityPicked;
            return (
              <tr key={item.id}>
                <td>{item.product.sku} - {item.product.name}</td>
                <td>{item.location.code}</td>
                <td>{item.quantityRequested}</td>
                <td>{item.quantityPicked}</td>
                <td>{item.status}</td>
                <td>
                  {item.status !== "PICKED" && (
                    <button onClick={() => handlePick(item.id, remaining)}>
                      Confirmar separação ({remaining})
                    </button>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
