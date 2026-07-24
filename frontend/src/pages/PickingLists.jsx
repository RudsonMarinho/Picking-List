import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { api } from "../lib/api";

export default function PickingLists() {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [pickingLists, setPickingLists] = useState([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api.get("/picking-lists", token).then(setPickingLists).catch((err) => setError(err.message));
  }, [token]);

  return (
    <div>
      <h1>Picking Lists</h1>
      {error && <p className="error">{error}</p>}

      <table>
        <thead>
          <tr>
            <th>Pedido</th>
            <th>Status</th>
            <th>Itens</th>
            <th>Criado em</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {pickingLists.map((pl) => (
            <tr key={pl.id}>
              <td>{pl.order.orderNumber}</td>
              <td>{pl.status}</td>
              <td>{pl.items.length}</td>
              <td>{new Date(pl.createdAt).toLocaleString("pt-BR")}</td>
              <td>
                <button onClick={() => navigate(`/picking-lists/${pl.id}`)}>Abrir</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
