import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  ApiError,
  AuthResponse,
  CartItem,
  Item,
  RegisterPayload,
  User,
  addToCart,
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
  const [addingItemID, setAddingItemID] = useState("");
  const [notice, setNotice] = useState("");

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
          setNotice(errorMessage(error));
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
    setNotice(`Добро пожаловать, ${response.user.first_name || response.user.username}!`);
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
    setNotice("");
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
      setNotice(`${item.name} добавлен в корзину`);
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

  function handlePrivateError(error: unknown) {
    if (error instanceof ApiError && error.status === 401) {
      clearSession();
      openAuth();
      setNotice("Сессия истекла. Войдите снова.");
      return;
    }
    setNotice(errorMessage(error));
  }

  return (
    <div className="market">
      <Header
        user={user}
        balance={balance}
        cartCount={cartCount}
        sessionLoading={sessionLoading}
        onLogin={() => openAuth("login")}
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

          {notice && (
            <div className="market-notice" role="status">
              {notice}
              <button onClick={() => setNotice("")} aria-label="Закрыть">
                <CloseIcon />
              </button>
            </div>
          )}

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

      {authOpen && (
        <AuthModal
          mode={authMode}
          onModeChange={setAuthMode}
          onClose={() => setAuthOpen(false)}
          onAuthenticated={handleAuthenticated}
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
  onCart: () => void;
  onLogout: () => void;
};

function Header({
  user,
  balance,
  cartCount,
  sessionLoading,
  onLogin,
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
            <div className="balance-pill">
              <span>Баланс</span>
              <strong>{balance === null ? "—" : currency.format(balance)}</strong>
            </div>
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
