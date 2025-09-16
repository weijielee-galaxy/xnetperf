ssh css-g88-001 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20030  10.155.84.3 &
ssh css-g88-001 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20031  10.155.84.3 &
ssh css-g88-002 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20032  10.155.84.3 &
ssh css-g88-002 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20033  10.155.84.3 &
ssh css-g88-004 ib_write_bw -d mlx5_0 --run_infinitely -m 4096 -p 20034  10.155.84.3 &
ssh css-g88-004 ib_write_bw -d mlx5_1 --run_infinitely -m 4096 -p 20035  10.155.84.3 &
