import { FeatureCollection } from "geojson";

export type PopupDataTypes = {
	isOpen: boolean;
	frequency: string;
	stationHeight: string;
};

export type SingleMapDataTypes = {
	stationPos: mapboxgl.LngLatLike,
	mapData: MapTypes,
	buildingsData: FeatureCollection,
	computationResult: mapboxgl.LngLatLike[][]
}  

export type MapTypes = {
    title: string;
    coordinates: number[][][];
    center: mapboxgl.LngLatLike;
    bounds: mapboxgl.LngLatBoundsLike;
};

export type MapTypesExtended = MapTypes & {
    stationPos: mapboxgl.LngLatLike;
    handleStationPosUpdate: (value: mapboxgl.LngLatLike) => void;
    buildingsData: GeoJSON.FeatureCollection | null
	computationResult: mapboxgl.LngLatLike[][]
}

export type PostComputeTypes = {
	freq: string;
	stationH: string;
};
