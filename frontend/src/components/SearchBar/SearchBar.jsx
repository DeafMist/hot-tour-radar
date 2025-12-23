import {useEffect, useState} from 'react';
import {useDebounce} from '../../hooks/useDebounce.js';
import './SearchBar.css';

export default function SearchBar({onSearch}) {
    const [query, setQuery] = useState('');
    const debouncedQuery = useDebounce(query, 600);

    useEffect(() => {
        onSearch(debouncedQuery);
    }, [debouncedQuery, onSearch]);

    return (
        <div className="search-container">
            <input
                className="search-input"
                type="text"
                placeholder="Search news..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
            />
        </div>
    );
}
