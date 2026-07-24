import { Routes, Route } from "react-router-dom";
import Layout from "./components/Layout";
import PrivateRoute from "./components/PrivateRoute";
import Login from "./pages/Login";
import Dashboard from "./pages/Dashboard";
import Products from "./pages/Products";
import Stock from "./pages/Stock";
import Orders from "./pages/Orders";
import PickingLists from "./pages/PickingLists";
import PickingListDetail from "./pages/PickingListDetail";

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        element={
          <PrivateRoute>
            <Layout />
          </PrivateRoute>
        }
      >
        <Route path="/" element={<Dashboard />} />
        <Route path="/products" element={<Products />} />
        <Route path="/stock" element={<Stock />} />
        <Route path="/orders" element={<Orders />} />
        <Route path="/picking-lists" element={<PickingLists />} />
        <Route path="/picking-lists/:id" element={<PickingListDetail />} />
      </Route>
    </Routes>
  );
}
