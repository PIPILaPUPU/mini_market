import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  ApiError,
  AuthResponse,
  CartItem,
  Item,
  RegisterPayload,
  User,
  addToCart,
  deposit,
  getBalance,
  getCart,
  getCurrentUser,
  getItems,
  login,
  logout as logoutRequest,
  register,
  removeFromCart,
} from "./api";

type AuthMode = "login" | "register";

type Toast = {
  id: string;
  message: string;
  tone: "success" | "error" | "info";
};

const currency = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "RUB",
  maximumFractionDigits: 2,
});

export default function App() {
  const [items, setItems] = useState<Item[]>([]);
  const [itemsLoading, setItemsLoading] = useState(true);
  const [itemsError, setItemsError] = useState("");
  const [token, setToken] = useState(
    () => sessionStorage.getItem("access_token") ?? "",
  );
  const [user, setUser] = useState<User | null>(null);
  const [balance, setBalance] = useState<number | null>(null);
  const [cart, setCart] = useState<CartItem[]>([]);
  const [sessionLoading, setSessionLoading] = useState(Boolean(token));
  const [authOpen, setAuthOpen] = useState(false);
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [cartOpen, setCartOpen] = useState(false);
  const [depositOpen, setDepositOpen] = useState(false);
  const [addingItemID, setAddingItemID] = useState("");
  const [toasts, setToasts] = useState<Toast[]>([]);

  useEffect(() => {
    let active = true;
    setItemsLoading(true);
    getItems()
      .then((result) => {
        if (active) {
          setItems(result);
          setItemsError("");
        }
      })
      .catch((error) => {
        if (active) {
          setItemsError(errorMessage(error));
        }
      })
      .finally(() => {
        if (active) {
          setItemsLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!token) {
      setSessionLoading(false);
      return;
    }

    let active = true;
    setSessionLoading(true);
    Promise.all([getCurrentUser(token), getBalance(token), getCart(token)])
      .then(([currentUser, wallet, currentCart]) => {
        if (!active) return;
        setUser(currentUser);
        setBalance(wallet.balance);
        setCart(currentCart);
      })
      .catch((error) => {
        if (!active) return;
        if (error instanceof ApiError && error.status === 401) {
          clearSession();
        } else {
          addToast(errorMessage(error), "error");
        }
      })
      .finally(() => {
        if (active) {
          setSessionLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [token]);

  const cartCount = useMemo(
    () => cart.reduce((total, position) => total + position.quantity, 0),
    [cart],
  );
  const cartTotal = useMemo(
    () =>
      cart.reduce(
        (total, position) =>
          total + position.item.price * position.quantity,
        0,
      ),
    [cart],
  );

  function clearSession() {
    sessionStorage.removeItem("access_token");
    setToken("");
    setUser(null);
    setBalance(null);
    setCart([]);
    setCartOpen(false);
    setDepositOpen(false);
  }

  function openAuth(mode: AuthMode = "login") {
    setAuthMode(mode);
    setAuthOpen(true);
  }

  function handleAuthenticated(response: AuthResponse) {
    sessionStorage.setItem("access_token", response.access_token);
    setUser(response.user);
    setToken(response.access_token);
    setAuthOpen(false);
    addToast(
      `Добро пожаловать, ${response.user.first_name || response.user.username}!`,
      "success",
    );
  }

  function addToast(message: string, tone: Toast["tone"] = "info") {
    const id = `${Date.now()}-${Math.random()}`;
    setToasts((current) => [...current, { id, message, tone }]);
    window.setTimeout(() => {
      setToasts((current) => current.filter((toast) => toast.id !== id));
    }, 5000);
  }

  function removeToast(id: string) {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }

  async function handleLogout() {
    try {
      await logoutRequest();
    } catch {
      // Локальную сессию нужно завершить, даже если refresh-cookie уже истекла.
    } finally {
      clearSession();
    }
  }

  async function handleAddToCart(item: Item) {
    if (!token || !user) {
      openAuth();
      return;
    }

    setAddingItemID(item.id);
    try {
      const position = await addToCart(token, item.id);
      setCart((current) => {
        const exists = current.some(
          (cartItem) => cartItem.item.id === position.item.id,
        );
        return exists
          ? current.map((cartItem) =>
              cartItem.item.id === position.item.id ? position : cartItem,
            )
          : [position, ...current];
      });
      addToast(`${item.name} добавлен в корзину`, "success");
    } catch (error) {
      handlePrivateError(error);
    } finally {
      setAddingItemID("");
    }
  }

  async function handleRemoveFromCart(itemID: string) {
    if (!token) return;
    try {
      await removeFromCart(token, itemID);
      setCart((current) =>
        current.filter((position) => position.item.id !== itemID),
      );
    } catch (error) {
      handlePrivateError(error);
    }
  }

  async function handleDeposit(
    amount: number,
    card: {
      card_number: string;
      exp_month: number;
      exp_year: number;
      cvv: string;
    },
  ): Promise<number> {
    if (!token) {
      throw new ApiError("Сессия истекла", 401, "invalid_token");
    }
    try {
      const result = await deposit(token, { amount, card });
      setBalance(result.balance);
      return result.balance;
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        handlePrivateError(error);
      }
      throw error;
    }
  }

  function handlePrivateError(error: unknown) {
    if (error instanceof ApiError && error.status === 401) {
      clearSession();
      openAuth();
      addToast("Сессия истекла. Войдите снова.", "error");
      return;
    }
    addToast(errorMessage(error), "error");
  }

  return (
    <div className="market">
      <Header
        user={user}
        balance={balance}
        cartCount={cartCount}
        sessionLoading={sessionLoading}
        onLogin={() => openAuth("login")}
        onDeposit={() => setDepositOpen(true)}
        onCart={() => setCartOpen(true)}
        onLogout={handleLogout}
      />

      <main>
        <section className="hero">
          <div className="hero__content">
            <p className="eyebrow">Mini market</p>
            <h1>Находки для жизни, собранные в одном месте.</h1>
            <p>
              Откройте каталог, выберите нужное и добавьте в корзину. Смотреть
              товары можно без регистрации.
            </p>
            <a className="hero__action" href="#catalog">
              Смотреть товары <ArrowIcon />
            </a>
          </div>
          <div className="hero__visual" aria-hidden="true">
            <span className="hero__ring hero__ring--outer" />
            <span className="hero__ring hero__ring--inner" />
            <span className="hero__monogram">M</span>
          </div>
        </section>

        <section className="catalog" id="catalog">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Каталог</p>
              <h2>Популярные товары</h2>
            </div>
            {!itemsLoading && !itemsError && (
              <span>{items.length} товаров</span>
            )}
          </div>

          {itemsLoading && <ProductSkeletons />}
          {!itemsLoading && itemsError && (
            <div className="empty-state">
              <h3>Не удалось загрузить каталог</h3>
              <p>{itemsError}</p>
              <button onClick={() => window.location.reload()}>
                Попробовать снова
              </button>
            </div>
          )}
          {!itemsLoading && !itemsError && items.length === 0 && (
            <div className="empty-state">
              <h3>Каталог пока пуст</h3>
              <p>Товары появятся здесь после загрузки образцов.</p>
            </div>
          )}
          {!itemsLoading && !itemsError && items.length > 0 && (
            <div className="product-grid">
              {items.map((item) => (
                <ProductCard
                  key={item.id}
                  item={item}
                  loading={addingItemID === item.id}
                  onAdd={() => handleAddToCart(item)}
                />
              ))}
            </div>
          )}
        </section>
      </main>

      <ToastStack toasts={toasts} onClose={removeToast} />

      {authOpen && (
        <AuthModal
          mode={authMode}
          onModeChange={setAuthMode}
          onClose={() => setAuthOpen(false)}
          onAuthenticated={handleAuthenticated}
        />
      )}

      {depositOpen && (
        <DepositModal
          onClose={() => setDepositOpen(false)}
          onDeposit={handleDeposit}
        />
      )}

      <CartDrawer
        open={cartOpen}
        items={cart}
        total={cartTotal}
        onClose={() => setCartOpen(false)}
        onRemove={handleRemoveFromCart}
      />
    </div>
  );
}

type HeaderProps = {
  user: User | null;
  balance: number | null;
  cartCount: number;
  sessionLoading: boolean;
  onLogin: () => void;
  onDeposit: () => void;
  onCart: () => void;
  onLogout: () => void;
};

function Header({
  user,
  balance,
  cartCount,
  sessionLoading,
  onLogin,
  onDeposit,
  onCart,
  onLogout,
}: HeaderProps) {
  return (
    <header className="site-header">
      <a className="brand" href="/" aria-label="Mini Market">
        <span className="brand__mark" aria-hidden="true">
          M
        </span>
        <span>mini market</span>
      </a>

      <nav className="header-actions" aria-label="Пользовательское меню">
        {sessionLoading ? (
          <span className="session-loader">Проверяем сессию…</span>
        ) : user ? (
          <>
            <button
              className="balance-pill"
              onClick={onDeposit}
              title="Пополнить баланс"
            >
              <span>Баланс · пополнить</span>
              <strong>{balance === null ? "—" : currency.format(balance)}</strong>
            </button>
            <button className="cart-button" onClick={onCart}>
              <CartIcon />
              <span>Корзина</span>
              {cartCount > 0 && <b>{cartCount}</b>}
            </button>
            <div className="user-chip">
              <span>{(user.first_name || user.username).slice(0, 1).toUpperCase()}</span>
              <div>
                <strong>{user.first_name || user.username}</strong>
                <button onClick={onLogout}>Выйти</button>
              </div>
            </div>
          </>
        ) : (
          <button className="login-button" onClick={onLogin}>
            <UserIcon />
            Войти
          </button>
        )}
      </nav>
    </header>
  );
}

function ProductCard({
  item,
  loading,
  onAdd,
}: {
  item: Item;
  loading: boolean;
  onAdd: () => void;
}) {
  const [imageFailed, setImageFailed] = useState(false);

  return (
    <article className="product-card">
      <div className="product-card__image">
        {item.image_url && !imageFailed ? (
          <img
            src={item.image_url}
            alt={item.name}
            loading="lazy"
            onError={() => setImageFailed(true)}
          />
        ) : (
          <ImagePlaceholder />
        )}
      </div>
      <div className="product-card__body">
        <h3>{item.name}</h3>
        <p>{item.description}</p>
        <div className="product-card__footer">
          <strong>{currency.format(item.price)}</strong>
          <button onClick={onAdd} disabled={loading}>
            <CartPlusIcon />
            {loading ? "Добавляем…" : "В корзину"}
          </button>
        </div>
      </div>
    </article>
  );
}

function AuthModal({
  mode,
  onModeChange,
  onClose,
  onAuthenticated,
}: {
  mode: AuthMode;
  onModeChange: (mode: AuthMode) => void;
  onClose: () => void;
  onAuthenticated: (response: AuthResponse) => void;
}) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [registration, setRegistration] = useState<RegisterPayload>({
    username: "",
    email: "",
    password: "",
    first_name: "",
    last_name: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [info, setInfo] = useState("");

  useEffect(() => {
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [onClose]);

  function switchMode(nextMode: AuthMode) {
    setError("");
    setInfo("");
    onModeChange(nextMode);
  }

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      onAuthenticated(await login(username, password));
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }

  async function handleRegister(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      await register(registration);
      await logoutRequest().catch(() => undefined);
      setUsername(registration.username);
      setPassword("");
      onModeChange("login");
      setInfo("Аккаунт создан. Теперь войдите.");
    } catch (requestError) {
      setError(errorMessage(requestError));
    } finally {
      setLoading(false);
    }
  }

  function updateRegistration(field: keyof RegisterPayload, value: string) {
    setRegistration((current) => ({ ...current, [field]: value }));
  }

  return (
    <div className="modal-backdrop" onMouseDown={onClose}>
      <section
        className="auth-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="auth-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <button className="icon-button auth-modal__close" onClick={onClose}>
          <CloseIcon />
          <span className="sr-only">Закрыть</span>
        </button>

        <div className="auth-modal__brand">
          <span className="brand__mark">M</span>
          <div>
            <p className="eyebrow">Mini market</p>
            <strong>Покупайте с комфортом</strong>
          </div>
        </div>

        <div className="auth-tabs" role="tablist">
          <button
            className={mode === "login" ? "active" : ""}
            onClick={() => switchMode("login")}
          >
            Вход
          </button>
          <button
            className={mode === "register" ? "active" : ""}
            onClick={() => switchMode("register")}
          >
            Регистрация
          </button>
        </div>

        {mode === "login" ? (
          <>
            <div className="form-heading">
              <h2 id="auth-title">Войдите в аккаунт</h2>
              <p>Чтобы открыть баланс и пользоваться корзиной.</p>
            </div>
            {info && <div className="alert alert--info">{info}</div>}
            {error && <div className="alert alert--error">{error}</div>}
            <form className="auth-form" onSubmit={handleLogin}>
              <AuthInput
                label="Имя пользователя"
                value={username}
                onChange={setUsername}
                autoComplete="username"
                placeholder="Ваш username"
                icon={<UserIcon />}
              />
              <AuthInput
                label="Пароль"
                value={password}
                onChange={setPassword}
                autoComplete="current-password"
                placeholder="Введите пароль"
                type="password"
                icon={<LockIcon />}
              />
              <button className="primary-button" disabled={loading}>
                {loading ? "Входим…" : "Войти"}
                {!loading && <ArrowIcon />}
              </button>
            </form>
          </>
        ) : (
          <>
            <div className="form-heading">
              <h2 id="auth-title">Создайте аккаунт</h2>
              <p>Регистрация займёт меньше минуты.</p>
            </div>
            {error && <div className="alert alert--error">{error}</div>}
            <form className="auth-form" onSubmit={handleRegister}>
              <div className="form-row">
                <AuthInput
                  label="Имя"
                  value={registration.first_name}
                  onChange={(value) => updateRegistration("first_name", value)}
                  autoComplete="given-name"
                  placeholder="Иван"
                />
                <AuthInput
                  label="Фамилия"
                  value={registration.last_name}
                  onChange={(value) => updateRegistration("last_name", value)}
                  autoComplete="family-name"
                  placeholder="Иванов"
                />
              </div>
              <AuthInput
                label="Имя пользователя"
                value={registration.username}
                onChange={(value) => updateRegistration("username", value)}
                autoComplete="username"
                placeholder="ivan_ivanov"
                pattern="[A-Za-z0-9_]{3,32}"
                icon={<UserIcon />}
              />
              <AuthInput
                label="Email"
                value={registration.email}
                onChange={(value) => updateRegistration("email", value)}
                autoComplete="email"
                placeholder="name@example.com"
                type="email"
                icon={<MailIcon />}
              />
              <AuthInput
                label="Пароль"
                value={registration.password}
                onChange={(value) => updateRegistration("password", value)}
                autoComplete="new-password"
                placeholder="Минимум 8 символов"
                type="password"
                minLength={8}
                maxLength={72}
                icon={<LockIcon />}
              />
              <button className="primary-button" disabled={loading}>
                {loading ? "Создаём аккаунт…" : "Зарегистрироваться"}
                {!loading && <ArrowIcon />}
              </button>
            </form>
          </>
        )}
      </section>
    </div>
  );
}

function AuthInput({
  label,
  value,
  onChange,
  icon,
  type = "text",
  ...inputProps
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  icon?: React.ReactNode;
  type?: string;
  autoComplete?: string;
  placeholder?: string;
  pattern?: string;
  minLength?: number;
  maxLength?: number;
}) {
  return (
    <label>
      <span>{label}</span>
      <span className="input-wrap">
        {icon}
        <input
          {...inputProps}
          type={type}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          required
        />
      </span>
    </label>
  );
}

function DepositModal({
  onClose,
  onDeposit,
}: {
  onClose: () => void;
  onDeposit: (
    amount: number,
    card: {
      card_number: string;
      exp_month: number;
      exp_year: number;
      cvv: string;
    },
  ) => Promise<number>;
}) {
  const [amount, setAmount] = useState("");
  const [cardNumber, setCardNumber] = useState("");
  const [expMonth, setExpMonth] = useState("");
  const [expYear, setExpYear] = useState("");
  const [cvv, setCVV] = useState("");
  const [phase, setPhase] = useState<"form" | "processing" | "success">(
    "form",
  );
  const [newBalance, setNewBalance] = useState<number | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape" && phase !== "processing") onClose();
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [onClose, phase]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    const parsedAmount = Number(amount);
    const month = Number(expMonth);
    const year = Number(expYear);
    const normalizedNumber = cardNumber.replace(/\s/g, "");

    if (!Number.isFinite(parsedAmount) || parsedAmount <= 0) {
      setError("Введите положительную сумму пополнения.");
      return;
    }
    if (normalizedNumber.length < 13) {
      setError("Проверьте номер карты.");
      return;
    }
    if (month < 1 || month > 12 || expYear.length !== 4) {
      setError("Укажите корректный месяц и год.");
      return;
    }
    if (cvv.length !== 3) {
      setError("CVV должен состоять из трёх цифр.");
      return;
    }

    setPhase("processing");
    const [paymentResult] = await Promise.allSettled([
      onDeposit(parsedAmount, {
        card_number: normalizedNumber,
        exp_month: month,
        exp_year: year,
        cvv,
      }),
      new Promise((resolve) => window.setTimeout(resolve, 3000)),
    ]);

    if (paymentResult.status === "rejected") {
      setPhase("form");
      setError(depositErrorMessage(paymentResult.reason));
      return;
    }

    setNewBalance(paymentResult.value);
    setPhase("success");
  }

  function changeCardNumber(value: string) {
    const digits = value.replace(/\D/g, "").slice(0, 19);
    setCardNumber(digits.match(/.{1,4}/g)?.join(" ") ?? "");
  }

  return (
    <div
      className="modal-backdrop"
      onMouseDown={phase === "processing" ? undefined : onClose}
    >
      <section
        className="auth-modal deposit-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="deposit-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        {phase !== "processing" && (
          <button className="icon-button auth-modal__close" onClick={onClose}>
            <CloseIcon />
            <span className="sr-only">Закрыть</span>
          </button>
        )}

        {phase === "processing" ? (
          <div className="payment-status payment-status--processing">
            <div className="payment-spinner" aria-hidden="true">
              <span />
              <CardIcon />
            </div>
            <p className="eyebrow">Безопасный платёж</p>
            <h2 id="deposit-title">Обрабатываем пополнение</h2>
            <p>Проверяем данные карты и подтверждаем операцию…</p>
            <div className="processing-progress" aria-hidden="true">
              <span />
            </div>
          </div>
        ) : phase === "success" ? (
          <div className="payment-status payment-status--success">
            <div className="success-mark" aria-hidden="true">
              <CheckIcon />
            </div>
            <p className="eyebrow">Готово</p>
            <h2 id="deposit-title">Баланс успешно пополнен</h2>
            <p>
              На кошелёк зачислено <strong>{currency.format(Number(amount))}</strong>
            </p>
            {newBalance !== null && (
              <div className="new-balance">
                <span>Новый баланс</span>
                <strong>{currency.format(newBalance)}</strong>
              </div>
            )}
            <button className="primary-button" onClick={onClose}>
              Готово
            </button>
          </div>
        ) : (
          <>
          <div className="form-heading deposit-heading">
          <p className="eyebrow">Кошелёк</p>
          <h2 id="deposit-title">Пополнить баланс</h2>
          <p>Данные карты проверяются и не сохраняются.</p>
        </div>

        <div className="card-preview" aria-hidden="true">
          <div className="card-preview__top">
            <span>MINI MARKET</span>
            <ContactlessIcon />
          </div>
          <strong>{cardNumber || "•••• •••• •••• ••••"}</strong>
          <div>
            <span>VALID THRU</span>
            <b>
              {expMonth.padStart(2, "0") || "MM"}/
              {expYear ? expYear.slice(-2) : "YY"}
            </b>
          </div>
        </div>

        {error && (
          <div className="alert alert--error" role="alert">
            {error}
          </div>
        )}

        <form className="auth-form deposit-form" onSubmit={handleSubmit}>
          <label>
            <span>Сумма пополнения</span>
            <span className="input-wrap amount-input">
              <input
                type="number"
                inputMode="decimal"
                min="1"
                step="0.01"
                value={amount}
                onChange={(event) => setAmount(event.target.value)}
                placeholder="1000"
                required
              />
              <b>₽</b>
            </span>
          </label>

          <div className="amount-presets" aria-label="Быстрый выбор суммы">
            {[500, 1000, 3000].map((value) => (
              <button type="button" key={value} onClick={() => setAmount(String(value))}>
                + {currency.format(value)}
              </button>
            ))}
          </div>

          <label>
            <span>Номер карты</span>
            <span className="input-wrap">
              <CardIcon />
              <input
                inputMode="numeric"
                autoComplete="cc-number"
                value={cardNumber}
                onChange={(event) => changeCardNumber(event.target.value)}
                placeholder="4242 4242 4242 4242"
                minLength={15}
                required
              />
            </span>
          </label>

          <div className="card-details-row">
            <label>
              <span>Месяц</span>
              <span className="input-wrap">
                <input
                  inputMode="numeric"
                  autoComplete="cc-exp-month"
                  value={expMonth}
                  onChange={(event) =>
                    setExpMonth(event.target.value.replace(/\D/g, "").slice(0, 2))
                  }
                  placeholder="MM"
                  required
                />
              </span>
            </label>
            <label>
              <span>Год</span>
              <span className="input-wrap">
                <input
                  inputMode="numeric"
                  autoComplete="cc-exp-year"
                  value={expYear}
                  onChange={(event) =>
                    setExpYear(event.target.value.replace(/\D/g, "").slice(0, 4))
                  }
                  placeholder="2030"
                  required
                />
              </span>
            </label>
            <label>
              <span>CVV</span>
              <span className="input-wrap">
                <input
                  type="password"
                  inputMode="numeric"
                  autoComplete="cc-csc"
                  value={cvv}
                  onChange={(event) =>
                    setCVV(event.target.value.replace(/\D/g, "").slice(0, 3))
                  }
                  placeholder="•••"
                  required
                />
              </span>
            </label>
          </div>

          <p className="test-card-note">
            Для проверки используйте тестовую карту 4242 4242 4242 4242.
          </p>

          <button className="primary-button">
            {`Пополнить${Number(amount) > 0 ? ` на ${currency.format(Number(amount))}` : ""}`}
          </button>
        </form>
          </>
        )}
      </section>
    </div>
  );
}

function CartDrawer({
  open,
  items,
  total,
  onClose,
  onRemove,
}: {
  open: boolean;
  items: CartItem[];
  total: number;
  onClose: () => void;
  onRemove: (itemID: string) => void;
}) {
  useEffect(() => {
    if (!open) return;
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [open, onClose]);

  return (
    <div
      className={`drawer-layer ${open ? "drawer-layer--open" : ""}`}
      aria-hidden={!open}
    >
      <button
        className="drawer-backdrop"
        onClick={onClose}
        aria-label="Закрыть корзину"
        tabIndex={open ? 0 : -1}
      />
      <aside className="cart-drawer" aria-label="Корзина">
        <div className="cart-drawer__header">
          <div>
            <p className="eyebrow">Ваш заказ</p>
            <h2>Корзина</h2>
          </div>
          <button className="icon-button" onClick={onClose}>
            <CloseIcon />
            <span className="sr-only">Закрыть</span>
          </button>
        </div>

        {items.length === 0 ? (
          <div className="cart-empty">
            <CartIcon />
            <h3>Корзина пуста</h3>
            <p>Добавьте товары из каталога — они появятся здесь.</p>
            <button onClick={onClose}>Вернуться к покупкам</button>
          </div>
        ) : (
          <>
            <div className="cart-list">
              {items.map((position) => (
                <article className="cart-position" key={position.id}>
                  <div className="cart-position__image">
                    {position.item.image_url ? (
                      <img src={position.item.image_url} alt="" />
                    ) : (
                      <ImagePlaceholder />
                    )}
                  </div>
                  <div className="cart-position__content">
                    <h3>{position.item.name}</h3>
                    <span>
                      {position.quantity} × {currency.format(position.item.price)}
                    </span>
                    <strong>
                      {currency.format(position.quantity * position.item.price)}
                    </strong>
                  </div>
                  <button
                    className="remove-button"
                    onClick={() => onRemove(position.item.id)}
                    aria-label={`Удалить ${position.item.name}`}
                  >
                    <TrashIcon />
                  </button>
                </article>
              ))}
            </div>
            <div className="cart-summary">
              <div>
                <span>Итого</span>
                <strong>{currency.format(total)}</strong>
              </div>
              <button className="primary-button" disabled>
                Оформление скоро появится
              </button>
            </div>
          </>
        )}
      </aside>
    </div>
  );
}

function ToastStack({
  toasts,
  onClose,
}: {
  toasts: Toast[];
  onClose: (id: string) => void;
}) {
  return (
    <section className="toast-stack" aria-live="polite" aria-label="Уведомления">
      {toasts.map((toast) => (
        <article
          className={`toast toast--${toast.tone}`}
          role="status"
          key={toast.id}
        >
          <div className="toast__icon" aria-hidden="true">
            {toast.tone === "success" ? (
              <CheckIcon />
            ) : (
              <strong>{toast.tone === "error" ? "!" : "i"}</strong>
            )}
          </div>
          <p>{toast.message}</p>
          <button
            onClick={() => onClose(toast.id)}
            aria-label="Закрыть уведомление"
          >
            <CloseIcon />
          </button>
          <span className="toast__timer" aria-hidden="true" />
        </article>
      ))}
    </section>
  );
}

function ProductSkeletons() {
  return (
    <div className="product-grid" aria-label="Загрузка товаров">
      {Array.from({ length: 8 }, (_, index) => (
        <div className="product-card product-card--skeleton" key={index}>
          <span />
          <div>
            <i />
            <i />
            <i />
          </div>
        </div>
      ))}
    </div>
  );
}

function ImagePlaceholder() {
  return (
    <div className="image-placeholder" aria-hidden="true">
      <span>M</span>
    </div>
  );
}

function depositErrorMessage(error: unknown) {
  if (!(error instanceof ApiError)) {
    return "Не удалось пополнить баланс. Попробуйте ещё раз.";
  }
  switch (error.message) {
    case "invalid card number":
      return "Проверьте номер карты.";
    case "unsupported payment system":
      return "Поддерживаются карты Visa, Mastercard и Мир.";
    case "invalid expiration date":
      return "Укажите корректный срок действия карты.";
    case "card is expired":
      return "Срок действия карты истёк.";
    case "cvv must contain 3 digits":
      return "CVV должен состоять из трёх цифр.";
    case "amount must be positive":
      return "Сумма пополнения должна быть больше нуля.";
    default:
      return error.message;
  }
}

function errorMessage(error: unknown) {
  if (!(error instanceof ApiError)) {
    return "Произошла непредвиденная ошибка.";
  }
  switch (error.code) {
    case "invalid_credentials":
      return "Неверное имя пользователя или пароль.";
    case "username_exists":
      return "Это имя пользователя уже занято.";
    case "email_exists":
      return "Этот email уже зарегистрирован.";
    case "invalid_request":
      return "Проверьте правильность заполнения полей.";
    default:
      return error.message;
  }
}

function UserIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm7 8a7 7 0 0 0-14 0" />
    </svg>
  );
}

function LockIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <rect x="5" y="10" width="14" height="10" rx="2" />
      <path d="M8 10V7a4 4 0 0 1 8 0v3" />
    </svg>
  );
}

function MailIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="m4 7 8 6 8-6" />
    </svg>
  );
}

function CardIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="M3 10h18M7 15h3" />
    </svg>
  );
}

function ContactlessIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M8 8.5a5 5 0 0 1 0 7M11 5.5a9 9 0 0 1 0 13M5 11a1.5 1.5 0 0 1 0 2" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m5 12 4 4L19 6" />
    </svg>
  );
}

function ArrowIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 12h14m-5-5 5 5-5 5" />
    </svg>
  );
}

function CartIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M3 4h2l2.2 10.1a2 2 0 0 0 2 1.6h7.9a2 2 0 0 0 2-1.6L20.5 8H6" />
      <circle cx="10" cy="20" r="1" />
      <circle cx="18" cy="20" r="1" />
    </svg>
  );
}

function CartPlusIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M3 4h2l2.2 10.1a2 2 0 0 0 2 1.6h7.9a2 2 0 0 0 2-1.6L20.5 8H6" />
      <path d="M14 10V4m-3 3h6" />
    </svg>
  );
}

function CloseIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m6 6 12 12M18 6 6 18" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M4 7h16m-10 4v6m4-6v6M9 7l1-3h4l1 3m3 0-1 14H7L6 7" />
    </svg>
  );
}
