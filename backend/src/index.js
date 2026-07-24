require("dotenv").config();
const express = require("express");
const cors = require("cors");

const authRoutes = require("./routes/auth");
const productRoutes = require("./routes/products");
const locationRoutes = require("./routes/locations");
const stockRoutes = require("./routes/stock");
const orderRoutes = require("./routes/orders");
const pickingListRoutes = require("./routes/pickingLists");

const app = express();

app.use(cors());
app.use(express.json());

app.get("/health", (req, res) => res.json({ status: "ok" }));

app.use("/auth", authRoutes);
app.use("/products", productRoutes);
app.use("/locations", locationRoutes);
app.use("/stock", stockRoutes);
app.use("/orders", orderRoutes);
app.use("/picking-lists", pickingListRoutes);

app.use((err, req, res, next) => {
  console.error(err);
  res.status(500).json({ error: "Erro interno do servidor" });
});

const port = process.env.PORT || 3333;
app.listen(port, () => {
  console.log(`Picking List API rodando na porta ${port}`);
});
