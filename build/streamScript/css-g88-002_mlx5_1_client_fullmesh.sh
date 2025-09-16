ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20018  10.155.84.2 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20019  10.155.84.2 &
ssh css-g88-003 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20020  10.155.84.2 &
ssh css-g88-003 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20021  10.155.84.2 &
ssh css-g88-004 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20022  10.155.84.2 &
ssh css-g88-004 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20023  10.155.84.2 &
